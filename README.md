# Scriptoria

## Purpose

Scriptoria is a service that is deployed and will monitor a source store either
located either locally or in the "cloud". It will look for PDF files that have
changed. On detection of a new file or a file that has changed it will download
the files and perform OCR (Optical Character Recognition) on them and convert
any text to a Markdown file of the same name. It will then upload the file to
a configured destination store.

## Configuration

### Environment Settings

We use environment settings for the more senstive information.

```sh
export GOOGLE_SERVICE_KEY_FILE="<Google Service Key File location>"
export GOOGLE_WATCH_FOLDER_ID="<Google Drive folder ID to watch for changes>"
export GOOGLE_ARCHIVE_FOLDER_ID="<Google Drive folder ID to move processed documents to>"
export GOOGLE_WEBHOOK_URL="<Web hook URL to receive file watch notifications. Must be SSL>"

export MATHPIX_APP_ID="<Mathpix App ID>"
export MATHPIX_APP_KEY="<Mathpix App Key>"

export CHATGPT_API_KEY="<ChatGPT API Key>"

export DATABASE_URL="<PostgreSQL database connection URL>"
export PORT=<Port to run the web service on>
```

### Configuration File Settings

We use a configuration file for settings that are not sensitive in nature.

```json
{
  "source_store": "Google Drive",
  "dest_store": "Local",
  "attachments_location": "<file path to where the original PDF image should be copied>",
  "notes_location": "<file path where the markdown note should be copied>",
  "local_storage_path": "<file path to temp local storage>"
}
```
