# Cohesion Violations — C++ Example

## Problem

A class whose methods operate on disjoint sets of member variables has low cohesion — it's mixing unrelated responsibilities.

```cpp
class UserManager {
private:
    std::string name;
    std::string email;
    std::ofstream fileBackend;  // Used only by SaveUserData
    std::ostream* logWriter;    // Used only by LogActivity

public:
    // Uses name, email
    std::pair<std::string, std::string> GetUserInfo() const {
        return {name, email};
    }

    // Uses fileBackend only
    bool SaveUserData() {
        fileBackend << name << "," << email << std::endl;
        return fileBackend.good();
    }

    // Uses logWriter only
    void LogActivity(const std::string& msg) {
        if (logWriter) *logWriter << msg << std::endl;
    }
};
```

Here, `GetUserInfo()` works with `name` and `email`; `SaveUserData()` works only with `fileBackend`; `LogActivity()` works only with `logWriter`. The class is gluing three independent concerns together.

## Solution

Split into focused classes, each with one responsibility:

```cpp
class User {
private:
    std::string name;
    std::string email;

public:
    std::pair<std::string, std::string> GetUserInfo() const {
        return {name, email};
    }
};

class UserPersistence {
private:
    std::ofstream backend;
    User* user;

public:
    bool SaveUserData() {
        backend << user->GetUserInfo().first << ","
                << user->GetUserInfo().second << std::endl;
        return backend.good();
    }
};

class UserLogger {
private:
    std::ostream* writer;

public:
    void LogActivity(const std::string& msg) {
        if (writer) *writer << msg << std::endl;
    }
};
```

Now each class has high cohesion: `User` manages identity, `UserPersistence` manages file I/O, `UserLogger` manages logging. Each can be tested and modified independently.
