package gotth

import "os"

func createFile(path string) (*os.File, error) { return os.Create(path) }

func removeFile(path string) { _ = os.Remove(path) }
