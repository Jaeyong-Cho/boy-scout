# Cohesion Violations — Go Example

## Problem

A struct whose methods touch disjoint sets of fields has low cohesion — it's gluing unrelated responsibilities together.

```go
type UserManager struct {
	name        string
	email       string
	fileBackend *os.File  // Used only by SaveUserData
	logWriter   io.Writer // Used only by LogActivity
}

// Uses name, email
func (um *UserManager) GetUserInfo() (string, string) {
	return um.name, um.email
}

// Uses fileBackend only
func (um *UserManager) SaveUserData() error {
	_, err := um.fileBackend.WriteString(um.name + "," + um.email)
	return err
}

// Uses logWriter only
func (um *UserManager) LogActivity(msg string) error {
	_, err := um.logWriter.Write([]byte(msg))
	return err
}
```

Here, `GetUserInfo()` works with `name` and `email`; `SaveUserData()` works only with `fileBackend`; `LogActivity()` works only with `logWriter`. The three methods don't share state — the struct is really three unrelated concerns shoved together.

## Solution

Split into focused structs, each with one responsibility:

```go
type User struct {
	name  string
	email string
}

func (u *User) GetUserInfo() (string, string) {
	return u.name, u.email
}

type UserPersistence struct {
	backend *os.File
	user    *User
}

func (up *UserPersistence) SaveUserData() error {
	_, err := up.backend.WriteString(up.user.name + "," + up.user.email)
	return err
}

type UserLogger struct {
	writer io.Writer
}

func (ul *UserLogger) LogActivity(msg string) error {
	_, err := ul.writer.Write([]byte(msg))
	return err
}
```

Now each struct has high cohesion: `User` manages identity, `UserPersistence` manages disk I/O, `UserLogger` manages logging. Tests can create and test each independently.
