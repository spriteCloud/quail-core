package composer

import "os"

func readFileImpl(p string) ([]byte, error) { return os.ReadFile(p) }
