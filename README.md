# Paperless Mailservice

Paperless Mailservice is a simple Go application that pulls all documents marked with a custom tag and sends them to a defined email address. You need an SMTP mail server, a running Paperless instance, and any environment to run this Docker container.

## Table of Contents

- [Paperless Mailservice](#paperless-mailservice)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Setting up Paperless-ngx](#setting-up-paperless-ngx)
  - [Deployment](#deployment)
  - [Configuration](#configuration)
    - [Environment Variables](#environment-variables)
    - [Example Values](#example-values)
  - [Docker Compose](#docker-compose)
  - [Docker Image Registry](#docker-image-registry)

## Overview

The Go OpenSource Project is a simple, efficient, and scalable application built using the Go programming language. This project is designed to be easily deployable and configurable to meet your specific needs.

## Setting up Paperless-ngx

You need to have a running [Paperless-ngx](https://github.com/paperless-ngx/paperless-ngx) instance. Copy the API Auth Token from "Edit Profile." If the key is not available, consider adjusting the permissions for your account. Please keep in mind that you need to create two custom tags. One marks for future processing, and the other indicates that the document was already sent.

Keep in mind the tool only works if you have PDF documents available. The send mail function currently only supports PDF attachments.

## Deployment

To deploy the project simply, you need to have Docker and Docker Compose installed on your machine. Follow the steps below to get started:

1. Clone the repository:
   ```sh
   git clone https://github.com/carlosz1986/paperless-mailservice.git
   cd paperless-mailservice
   ```
2. Create a .env config file based on the sample:
   ```sh
   cp .env.example .env
   ```

3. Run the project using Docker Compose:
   ```sh
   docker-compose up
   ```

4. Optional: Build the project using Docker Compose:
   ```sh
   docker-compose up --build
   ```

5. Optional: If you want to run the binary standalone, compile the binary with:
   ```sh
   go run main.go
   ```

## Configuration

The project can be configured using environment variables. Below are the details on how to set up and configure these variables.

### Environment Variables

| Variable Name          | Description                                                                            | Example Value                          |
|------------------------|----------------------------------------------------------------------------------------|----------------------------------------|
| `paperlessInstanceURL` | The API Endpoint of the Paperless instance. Don't forget the / at the end.             | `http://192.168.178.48:8000/api/`      |
| `paperlessInstanceToken` | The Paperless API Token                                                               | `9d02951f3716e098b`                    |
| `processedTagName`     | The application assigns a tag to every processed document to prevent sending twice. Add the string of the tag name. | `DatevSent`                            |
| `searchTagName`        | The tag name used for searching documents e.g. marking them for sending.                                             | `SendToDatev`                          |
| `receiverEmail`        | Email address of the receiver                                                          | `receiverEmail`                        |
| `smtpEmail`            | Sender email address, which is also the username                                       | `mail.com`                             |
| `smtpServer`           | An SMTP mail server, with TLS or without                                               | `smtpServer`                           |
| `smtpPort`             | Port of the SMTP mail server                                                           | `587`                                  |
| `smtpConnectionType`   | SMTP Connection Type: If the Port is 587, normally starttls is correct. Otherwise tls. | `starttls` OR `tls`                                  |
| `smtpUser`             | SMTP Username                                                                          | `peter`                            |
| `smtpPassword`         | SMTP password                                                                          | `fQsdfsdfs`                            |
| `mailBody`             | A custom string that is added to the email body                                        | `You got a file ...`                   |
| `mailHeader`           | A custom string that is added to the email header                                      | `Header - file`                        |
| `runEveryXMinute`      | Minutes break between every execution. -1 starts the execution once                    | `1`                                    |

### Example Values

Put the .env file in the docker-compose.yaml folder. It will be consumed automatically on container start. Here is an example of how to set environment variables in your `.env` file:

```env
paperlessInstanceURL="http://192.168.178.48:8000/api/"
paperlessInstanceToken=9d02951f3716e098b
processedTagName=DatevSent
searchTagName=SendToDatev
receiverEmail=you@get.it
smtpEmail=bla@foo.bar
smtpServer=mail.com
smtpPort=587
smtpConnectionType=starttls
smtpUser=peter
smtpPassword=fQsdfsdfs
mailBody="You got a file ..."
mailHeader="You got a file"
runEveryXMinute=1
```

## Docker Compose

The project includes a `docker-compose.yml` file for easy deployment. Below is a basic configuration:

```yaml
version: "3.9"
services:
  paperless-mailservice:
    build:
      dockerfile: Dockerfile
      context: .
    image: carlosz1986/paperless-mailservice:latest
    volumes:
      - .:/app
    environment:
      paperlessInstanceURL: ${paperlessInstanceURL}
      paperlessInstanceToken: ${paperlessInstanceToken}
      processedTagName: ${processedTagName}
      searchTagName: ${searchTagName}
      receiverEmail: ${receiverEmail}
      smtpEmail: ${smtpEmail}
      smtpServer: ${smtpServer}
      smtpPort: ${smtpPort}
      smtpConnectionType: ${smtpConnectionType}
      smtpUser: ${smtpUser}
      smtpPassword: ${smtpPassword}
      mailBody: ${mailBody}
      mailHeader: ${mailHeader}
      runEveryXMinute: ${runEveryXMinute}
```

## Docker Image Registry

The Docker image for this project is available on Docker Hub. You can pull the image using the following command:

```sh
docker pull carlosz1986/paperless-mailservice:latest
```

Visit the [Docker Image Registry](https://hub.docker.com/r/carlosz1986/paperless-mailservice) for more details.

---

Thank you for using the Go Paperless Mailservice! If you encounter any issues or have questions, feel free to open an issue on GitHub. Contributions are welcome!