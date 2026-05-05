package envfile

import (
	"github.com/joho/godotenv"
)

const defaultEnvFile = "environments/.env"

// Load reads a single env file: environments/.env (relative to the process working directory).
// It does NOT override already-set env vars, so real runtime env wins.
func Load() []string {
	if err := godotenv.Load(defaultEnvFile); err != nil {
		return nil
	}
	return []string{defaultEnvFile}
}
