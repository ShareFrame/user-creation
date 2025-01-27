# **User Creation Service**

This repository contains the **User Creation Service**, a Go-based application designed to handle user registration, validation, and storage in a scalable environment. It integrates with AWS services, the AT Protocol, and other internal APIs to ensure seamless user onboarding.

---

## **Features**
- **User Registration**: Validates and registers users with unique handles.
- **Email Validation**: Ensures proper email formatting during user registration.
- **Handle Validation**: Supports domain appending and ensures no symbols in user IDs.
- **AWS Integration**:
  - **Secrets Manager**: Securely retrieves admin credentials.
  - **DynamoDB**: Stores user data for persistence.
- **AT Protocol Support**: Generates invite codes and registers users on the protocol.
- **Email Notifications**: Sends confirmation emails to users after registration.

---

## **Technologies Used**
- **Programming Language**: Go (Golang)
- **Cloud Platform**: AWS
  - AWS Lambda
  - AWS DynamoDB
  - AWS Secrets Manager
- **Third-party Services**: Resend (for email)
- **Protocols**: AT Protocol

---

## **Project Structure**
- /config: Configuration and environment loading.
- /internal: Internal packages for services like AT Protocol, DynamoDB, and email.
- /handlers: API handlers for processing user requests.
- /models: Data structures and models.

---

## **Contributing**
Contributions are welcome! Please follow these steps:
1. Fork the repository
2. Create a feature branch:
    ```bash
    git checkout -b feature/your-feature
    ```
3. Commit your changes and push to your fork
4. Create a pull request.
