graph TD;
    A[User] --> B[BackendService];
    B --> C[EtherealPublicNode];
    C --> B;
    B --> A;
