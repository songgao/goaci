package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyRegularFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}
	return nil
}

func copySymlink(src, dest string) error {
	symTarget, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if err := os.Symlink(symTarget, dest); err != nil {
		return err
	}
	return nil
}

func copyTree(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rootLess := path[len(src):]
		target := filepath.Join(dest, rootLess)
		mode := info.Mode()
		switch {
		case mode.IsDir():
			err := os.Mkdir(target, mode.Perm())
			if err != nil {
				return err
			}
		case mode.IsRegular():
			if err := copyRegularFile(path, target); err != nil {
				return err
			}
		case mode&os.ModeSymlink == os.ModeSymlink:
			if err := copySymlink(path, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unsupported node %q in assets, only regular files, directories and symlinks are supported.", path, mode.String())
		}
		return nil
	})
}

func replacePlaceholders(path string, placeholderMapping map[string]string) string {
	Debug("Processing path: ", path)
	newPath := path
	for placeholder, replacement := range placeholderMapping {
		newPath = strings.Replace(newPath, placeholder, replacement, -1)
	}
	Debug("Processed path: ", newPath)
	return newPath
}

func validateAsset(ACIAsset, localAsset string) error {
	if !filepath.IsAbs(ACIAsset) {
		return fmt.Errorf("Wrong ACI asset: '%v' - ACI asset has to be absolute path", ACIAsset)
	}
	if !filepath.IsAbs(localAsset) {
		return fmt.Errorf("Wrong local asset: '%v' - local asset has to be absolute path", localAsset)
	}
	fi, err := os.Stat(localAsset)
	if err != nil {
		return fmt.Errorf("Error stating %v: %v", localAsset, err)
	}
	if fi.Mode().IsDir() || fi.Mode().IsRegular() {
		return nil
	}
	return fmt.Errorf("Can't handle local asset %v - not a file, not a dir", fi.Name())
}

func PrepareAssets(assets []string, rootfs string, placeholderMapping map[string]string) error {
	for _, asset := range assets {
		splitAsset := filepath.SplitList(asset)
		if len(splitAsset) != 2 {
			return fmt.Errorf("Malformed asset option: '%v' - expected two absolute paths separated with %v", asset, ListSeparator())
		}
		ACIAsset := replacePlaceholders(splitAsset[0], placeholderMapping)
		localAsset := replacePlaceholders(splitAsset[1], placeholderMapping)
		if err := validateAsset(ACIAsset, localAsset); err != nil {
			return err
		}
		ACIAssetSubPath := filepath.Join(rootfs, filepath.Dir(ACIAsset))
		err := os.MkdirAll(ACIAssetSubPath, 0755)
		if err != nil {
			return fmt.Errorf("Failed to create directory tree for asset '%v': %v", asset, err)
		}
		err = copyTree(localAsset, filepath.Join(rootfs, ACIAsset))
		if err != nil {
			return fmt.Errorf("Failed to copy assets for '%v': %v", asset, err)
		}
	}
	return nil
}
