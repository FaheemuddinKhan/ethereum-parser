graph TD;
    A[User] --> B[BackendService];
    B --> C[StorageService];
    C --> B;
    B --> A;
