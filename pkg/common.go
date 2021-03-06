package pkg

import (
	"bytes"
	"embed"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"
)

func ExtractEmbedFile(f embed.FS, rootDir string, targetDir string) error {
	return fs.WalkDir(f, rootDir, func(path string, d fs.DirEntry, err error) error {
		rootDir = strings.TrimSuffix(rootDir, "/")
		if err != nil {
			return err
		}
		if path != "." && path != rootDir {
			pathTarget := fmt.Sprintf("%s/%s", targetDir, strings.TrimPrefix(path, fmt.Sprintf("%s/", rootDir)))
			if d.IsDir() {
				_ = os.MkdirAll(pathTarget, 0700)
			} else {
				bs, err := f.ReadFile(path)
				if err != nil {
					fmt.Println("ERROR:", err.Error())
					return err
				}
				_ = os.WriteFile(pathTarget, bs, 0600)
			}
		}
		return nil
	})
}

func CommandExec(command, workDir string) (string, string, error) {
	var err error
	errInfo := fmt.Sprintf("exec %s error", command)
	var strOut, strErr string

	execCmd := exec.Command("sh", "-c", command)
	execCmd.Dir = workDir

	prOut, pwOut := io.Pipe()
	prErr, pwErr := io.Pipe()
	execCmd.Stdout = pwOut
	execCmd.Stderr = pwErr

	rOut := io.TeeReader(prOut, os.Stdout)
	rErr := io.TeeReader(prErr, os.Stderr)

	err = execCmd.Start()
	if err != nil {
		err = fmt.Errorf("%s: exec start error: %s", errInfo, err.Error())
		return strOut, strErr, err
	}

	var bOut, bErr bytes.Buffer

	go func() {
		_, _ = io.Copy(&bOut, rOut)
	}()

	go func() {
		_, _ = io.Copy(&bErr, rErr)
	}()

	err = execCmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				err = fmt.Errorf("%s: exit status: %d", errInfo, status.ExitStatus())
			}
		} else {
			err = fmt.Errorf("%s: exec run error: %s", errInfo, err.Error())
			return strOut, strErr, err
		}
	}

	strOut = bOut.String()
	strErr = bErr.String()

	return strOut, strErr, err
}

func CheckRandomStringStrength(password string, length int, enableSpecialChar bool) error {
	var err error

	if len(password) < length {
		err = fmt.Errorf("password must at least %d charactors", length)
		return err
	}
	lowerChars := "abcdefghijklmnopqrstuvwxyz"
	upperChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberChars := "0123456789"
	specialChars := `~!@#$%^&*()_+-={}[]\|:";'<>?,./`
	lowerOK := strings.ContainsAny(password, lowerChars)
	upperOK := strings.ContainsAny(password, upperChars)
	numberOK := strings.ContainsAny(password, numberChars)
	specialOK := strings.ContainsAny(password, specialChars)
	if enableSpecialChar && !(lowerOK && upperOK && numberOK && specialOK) {
		err = fmt.Errorf("password must include lower upper case charactors and number and special charactors")
		return err
	} else if !enableSpecialChar && !(lowerOK && upperOK && numberOK) {
		err = fmt.Errorf("password must include lower upper case charactors and number")
		return err
	}

	return err
}

func RandomString(n int, enableSpecialChar bool, suffix string) string {
	var letter []rune
	lowerChars := "abcdefghijklmnopqrstuvwxyz"
	upperChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberChars := "0123456789"
	specialChars := `~!@#$%^&*()_+-={}[]\|:";'<>?,./`
	if enableSpecialChar {
		chars := fmt.Sprintf("%s%s%s%s", lowerChars, upperChars, numberChars, specialChars)
		letter = []rune(chars)
	} else {
		chars := fmt.Sprintf("%s%s%s", lowerChars, upperChars, numberChars)
		letter = []rune(chars)
	}
	var pwd string
	for {
		b := make([]rune, n)
		seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := range b {
			b[i] = letter[seededRand.Intn(len(letter))]
		}
		pwd = string(b)
		err := CheckRandomStringStrength(pwd, n, enableSpecialChar)
		if err == nil {
			break
		}
	}
	return fmt.Sprintf("%s%s", pwd, suffix)
}

func ValidateIpAddress(s string) error {
	var err error
	if net.ParseIP(s).To4() == nil {
		err = fmt.Errorf(`not ipv4 address`)
		return err
	}
	return err
}

func ValidateMinusNameID(s string) error {
	var err error
	RegExp := regexp.MustCompile(`^(([a-z])[a-z0-9]+(-[a-z0-9]+)+)$`)
	match := RegExp.MatchString(s)
	if !match {
		err = fmt.Errorf(`should include lower case and number, format should like "hello-world-no3"`)
		return err
	}
	return err
}

func ValidateMinus(s string) error {
	var err error
	RegExp := regexp.MustCompile(`^([^-]+)(-([^-]+))+$`)
	match := RegExp.MatchString(s)
	if !match {
		err = fmt.Errorf(`format should like "hello-world-no3"`)
		return err
	}
	arr := strings.Split(s, "-")
	for _, str := range arr {
		err = ValidateWithoutSpecialChars(str)
		if err != nil {
			return err
		}
	}
	return err
}

func ValidateLowCaseName(s string) error {
	var err error
	RegExp := regexp.MustCompile(`^([a-z])[a-z0-9]+$`)
	match := RegExp.MatchString(s)
	if !match {
		err = fmt.Errorf(`should include lower case and number, format should like "test1"`)
		return err
	}
	return err
}

func ValidateWithoutSpecialChars(s string) error {
	var err error
	chars := `~!@#$%^&*()_+-={}|[]\:;"'<>?,./ ` + "`"
	match := strings.ContainsAny(s, chars)
	if match {
		err = fmt.Errorf(`cannot contain special chars`)
		return err
	}
	return err
}

func YamlIndent(obj interface{}) ([]byte, error) {
	var err error
	var bs []byte
	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	err = yamlEncoder.Encode(&obj)
	if err != nil {
		return bs, err
	}
	bs = b.Bytes()

	return bs, err
}

func RemoveMapEmptyItems(m map[string]interface{}) map[string]interface{} {
	for k, v := range m {
		vv := reflect.ValueOf(v)
		switch v.(type) {
		case int, float32, float64:
			if vv.IsZero() {
				delete(m, k)
			}
		case bool:
			if vv.Bool() == false {
				delete(m, k)
			}
		case string:
			if vv.String() == "" {
				delete(m, k)
			}
		}
		if !vv.IsValid() {
			delete(m, k)
		}
		if vv.Kind() == reflect.Slice {
			if vv.Len() == 0 {
				delete(m, k)
			} else {
				var isMap bool
				var x []map[string]interface{}
				for i := 0; i < vv.Len(); i++ {
					vvv := reflect.ValueOf(vv.Index(i))
					if vvv.Kind() == reflect.Map {
						vm, ok := vv.Index(i).Interface().(map[string]interface{})
						if ok {
							isMap = true
							v3 := RemoveMapEmptyItems(vm)
							x = append(x, v3)
						}
					} else if vvv.Kind() == reflect.Struct {
						vm, ok := vv.Index(i).Interface().(map[string]interface{})
						if ok {
							isMap = true
							v3 := RemoveMapEmptyItems(vm)
							x = append(x, v3)
						}
					}
				}
				if isMap {
					m[k] = x
				}
			}
		}
		if vv.Kind() == reflect.Struct {
			v2 := RemoveMapEmptyItems(v.(map[string]interface{}))
			if len(v2) == 0 {
				delete(m, k)
			} else {
				m[k] = v2
			}
		} else if vv.Kind() == reflect.Map {
			v2 := RemoveMapEmptyItems(v.(map[string]interface{}))
			if len(v2) == 0 {
				delete(m, k)
			} else {
				m[k] = v2
			}
		}
	}
	m2 := m
	return m2
}
