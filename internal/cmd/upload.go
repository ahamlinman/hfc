package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/shelley"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload the latest binary to the container registry",
	Run:   runUpload,
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(cmd *cobra.Command, args []string) {
	outputPath, err := rootState.BinaryPath(rootConfig.Project.Name)
	if err != nil {
		log.Fatal(err)
	}

	stat, err := os.Stat(outputPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Fatal("must build a binary before uploading")
	case err != nil:
		log.Fatal(err)
	case !stat.Mode().IsRegular():
		log.Fatalf("%s is not a regular file", outputPath)
	}

	outputHash, err := fileSHA256(outputPath)
	if err != nil {
		log.Fatal(err)
	}

	repository := shelley.GetOrExit(shelley.
		Command(
			"aws", "ecr", "describe-repositories",
			"--repository-names", rootConfig.Repository.Name,
			"--query", "repositories[0].repositoryUri", "--output", "text",
		).
		Debug().
		Text())

	registry := strings.SplitN(repository, "/", 2)[0]
	image := repository + ":" + outputHash

	authenticated := shelley.GetOrExit(shelley.
		Command("go", "run", "go.alexhamlin.co/zeroimage@main", "check-auth", "--push", image).
		Debug().
		NoOutput().
		Successful())

	if !authenticated {
		shelley.ExitIfError(shelley.
			Command("aws", "ecr", "get-login-password").
			Debug().
			Pipe(
				"go", "run", "go.alexhamlin.co/zeroimage@main",
				"login", "--username", "AWS", "--password-stdin", registry,
			).
			Debug().
			Run())
	}

	shelley.ExitIfError(shelley.
		Command(
			"go", "run", "go.alexhamlin.co/zeroimage@main",
			"build", "--platform", "linux/arm64", "--push", image, outputPath,
		).
		Debug().
		Run())

	latestImagePath := rootState.Path("latest-image")
	if err := os.WriteFile(latestImagePath, []byte(image), 0644); err != nil {
		log.Fatal(err)
	}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
