# Paperless Mailservice

Paperless Mailservice is a simple Go application that pulls all documents marked with a custom tag and sends them to a defined email address. You can define different rulesets that allows you to send docs with different tags to different email adresses. Besides you can also send one document to two different email adresses. You need an SMTP mail server, a running Paperless instance, and any environment to run this Docker container.

## Table of Contents

- [Paperless Mailservice](#paperless-mailservice)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Setting up Paperless-ngx](#setting-up-paperless-ngx)
  - [Deployment](#deployment)
  - [Configuration](#configuration)
    - [Yaml Config Variables](#yaml-config-variables)
    - [Placeholders for the Email Header and Body](#placeholders-for-the-email-header-and-body)
    - [Yaml Example Values](#yaml-example-values)
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
2. Create a config.yaml config file based on the sample:
   ```sh
   cd ./config && cp config.yaml.example config.yaml && cd ..
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

The project can be configured using a yaml config file named config.yaml. The file needs to be placed in `./config/` or has to be mounted into the container by using a volume. Below are the details on how to set up and configure these variables.

### Yaml Config Variables

| Type | Variable Name          | Description                                                                            | Example Value                          |
|------|------------------------|----------------------------------------------------------------------------------------|----------------------------------------|
| `Paperless` | `InstanceURL` | The base Endpoint of the Paperless instance. Don't forget the / at the end.                   | `http://192.168.178.48:8000/`      |
| `Paperless` | `InstanceToken` | The Paperless API Token                                                               | `9d02951f3716e098b`                    |
| `Paperless` | `ProcessedTagName`     | The application assigns a tag to every processed document to prevent sending twice. Add the string of the tag name. | `DatevSent`                            |
| `Paperless` | `SearchTagName`        | The tag name used for searching documents e.g. marking them for sending.                                             | `SendToDatev`                          |
| `Paperless` | `ReceiverAddress`        | The tag name used for searching documents e.g. marking them for sending.                                             | `SendToDatev`                          |
| `Paperless.Rules[]` | `Name`            | Custom Rule Name                                       | `OneDemoRule` |
| `Paperless.Rules[]` | `ReceiverAddress`            | Email address of the receiver                                        | `you@get.it`                             |
| `Paperless.Rules[]` | `MailHeader`            | A custom string that is added to the email header. If set it will overwrite the default Email.MailHeader.                              | `"Custom Header - file from %first_name%"`                             |
| `Paperless.Rules[]` | `MailBody`            | A custom string that is added to the email body.MailBody. If set it will overwrite the default Email.MailBody. HTML tags are supported.                              | `"You got a file: %document_title%. Open it <a href='%document_url%'>%document_id%</a>"`                             |
| `Paperless.Rules.Tags[]` | Keys            | Each Tag of that rule is one line, Tags are && linked                                        | `Invoices`                             |
| `Email` | `SMTPServer`           | An SMTP mail server, with TLS or without                                               | `smtpServer`                           |
| `Email` | `SMTPPort`             | Port of the SMTP mail server                                                           | `587`                                  |
| `Email` | `SMTPConnectionType`   | SMTP Connection Type: If the Port is 587, normally starttls is correct. Otherwise tls. | `starttls` OR `tls`                                  |
| `Email` | `SMTPUser`             | SMTP Username                                                                          | `peter`                            |
| `Email` | `SMTPPassword`         | SMTP password                                                                          | `fQsdfsdfs`                            |
| `Email` | `MailBody`             | A string that is added to the email body. HTML tags are supported.                     | `You got a file ...`                   |
| `Email` | `MailHeader`           | A string that is added to the email header.                                      | `Header - file`                        |
| `General` | `RunEveryXMinute`      | Minutes break between every execution. -1 starts the execution once                    | `1`                                    |

### Placeholders for the Email Header and Body

You can use different placeholders in the Header and Body configuration values. These values ​​will be replaced for each document when it is sent.

| Variable Name          | Description                                                                            |
|------------------------|----------------------------------------------------------------------------------------|
| `%user_id%` | The paperless Username ID by the document owner |
| `%user_name%` | The paperless Username by the document owner |
| `%user_email%` | The paperless Username Email Address by the document owner |
| `%first_name%` | The First Name by the document owner |
| `%last_name%` | The Last Name by the document owner |
| `%correspondent_name%` | The Correspdondent Name of the document |
| `%storage_path_id%` | The storage path ID of the document |
| `%storage_path%` | The storage path of the document |
| `%storage_path_name%` | The storage path name of the document |
| `%document_id%` | The Document ID |
| `%document_url%` | The http URL, that opens the document in paperless |
| `%document_title%` | The Documents title |
| `%document_type_ID%` | The Document Type ID |
| `%document_type_name%` | The Document Type Name |
| `%document_file_name%` | The Document Filename Name |
| `%document_created_at%` | The Date when the document was created |
| `%document_modified_at%` | The Date when the document was modified the last time |

### Yaml Example Values

Put the config.yaml file in the ./config/ folder. It will be consumed automatically on container start. Here is an example of how the `config.yaml` file should look like:

```yaml
Paperless:
  InstanceURL: http://192.168.178.48:8000/
  InstanceToken: 9d02951f3716e098b
  ProcessedTagName: DatevSent
  AddQueueTagName: SendToDatev
  Rules:
    - Name: "OneDemoRule"
      Tags: #The Doc must hold all three tags 
        - Seaside Docs
        - Invoices
        - Foobar
      ReceiverAddress: you@get.it
    - Name: "TwoDemoRule"
      Tags: # You can create mutiple rules for a Tag combination to send the doc to different receivers
        - OfflineDocs
      ReceiverAddress: dont@get.it
      #If Header and/or Body are set, the base Email.MailBody and/or Email.MailHeader will be overwritten.
      MailBody: "Custom Body for that rule - %first_name%"
      MailHeader: "Custom Header for that rule - %document_id%"
Email:
  SMTPAddress: bla@foo.bar
  SMTPServer: mail.com
  SMTPPort: 587
  SMTPConnectionType: starttls
  SMTPUser: bla@foo.bar
  SMTPPassword: fQsdfsdfs
  MailBody: "You got a file ...with some values %document_file_name%, %document_created_at%"
  MailHeader: "You got a file - %document_file_name%"
RunEveryXMinute: 1
```

## Docker Compose

The project includes a `docker-compose.yml` file for easy deployment. Below is a basic configuration:

```yaml
services:
  paperless-mailservice:
    build:
      dockerfile: Dockerfile
      context: .
    image: carlosz1986/paperless-mailservice:latest
    volumes:
      - .:/app
      - ./config:/config
```

## Docker Image Registry

The Docker image for this project is available on Docker Hub. You can pull the image using the following command:

```sh
docker pull carlosz1986/paperless-mailservice:latest
```

Visit the [Docker Image Registry](https://hub.docker.com/r/carlosz1986/paperless-mailservice) for more details.

---

Thank you for using the Go Paperless Mailservice! If you encounter any issues or have questions, feel free to open an issue on GitHub. Contributions are welcome!