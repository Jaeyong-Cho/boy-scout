# Cohesion Violations — TypeScript Example

## Problem

A class whose methods operate on disjoint sets of member variables has low cohesion — it's mixing unrelated responsibilities.

```typescript
class UserManager {
    private name: string;
    private email: string;
    private fileBackend: fs.WriteStream;  // Used only by SaveUserData
    private logWriter: fs.WriteStream;    // Used only by LogActivity

    // Uses name, email
    getUserInfo(): [string, string] {
        return [this.name, this.email];
    }

    // Uses fileBackend only
    saveUserData(): void {
        this.fileBackend.write(`${this.name},${this.email}\n`);
    }

    // Uses logWriter only
    logActivity(msg: string): void {
        this.logWriter.write(`${msg}\n`);
    }
}
```

Here, `getUserInfo()` works with `name` and `email`; `saveUserData()` works only with `fileBackend`; `logActivity()` works only with `logWriter`. The class glues three independent concerns together.

## Solution

Split into focused classes, each with one responsibility:

```typescript
class User {
    constructor(private name: string, private email: string) {}

    getUserInfo(): [string, string] {
        return [this.name, this.email];
    }
}

class UserPersistence {
    constructor(private backend: fs.WriteStream, private user: User) {}

    saveUserData(): void {
        const [name, email] = this.user.getUserInfo();
        this.backend.write(`${name},${email}\n`);
    }
}

class UserLogger {
    constructor(private writer: fs.WriteStream) {}

    logActivity(msg: string): void {
        this.writer.write(`${msg}\n`);
    }
}
```

Now each class has high cohesion: `User` manages identity, `UserPersistence` manages file I/O, `UserLogger` manages logging. Each can be tested and evolved independently.
