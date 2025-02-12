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

export MATHPIX_APP_ID="<Mathpix App ID>"
export MATHPIX_APP_KEY="<Mathpix App Key>"

export CHATGPT_API_KEY="<ChatGPT API Key>"

export DATABASE_URL="<PostgreSQL database connection URL>"
export PORT=<Port to run the web service on>
```

### Configuration File Settings

We use a configuration file for non-sensitive settings

```json
{
  "temp_storage_folder": "<temp folder for documents>",
  "source_store": "Google Drive",
  "bundles": [
    {
      "source_folder": "<Google Drive folder ID>",
      "archive_folder": "<Google Drive folder ID>",
      "dest_attachments_folder": "<local folder to copy original PDF to>",
      "dest_notes_folder": "<local folder to copy Markdown file to>"
    },
    {
      "source_folder": "<Google Drive folder ID>",
      "archive_folder": "<Google Drive folder ID>",
      "dest_attachments_folder": "<local folder to copy original PDF to>",
      "dest_notes_folder": "<local folder to copy Markdown file to>"
    }
  ]
}
```

- `temp_storage_folder` this is a local file folder that can be used by processors to stage the file.
- `source_store` currently we only support Google Drive. This would allow for future source storage locations to be monitored.
- `bundles` list of source folder and destination folders that are paired together. More on processing below.
- `bundles.source_folder` the source folder in the `source_store` to monitor for new files to process.
- `bundles.archive_folder` the folder to copy documents to once they are successfully processed.
- `bundles.dest_attachments_folder` the destination folder for the original PDF file that will be linked in the resulting Markdown.
- `bundles.dest_notes_folder` the destination folder for the resulting Markdown file.

### Storage

At this time only Google Drive is supported. Scriptoria will monitor a Google Drive folder location that is specified in the `bundles.source_folder` for any new files added.

### Processing

Processors are configured in a chain using channels. Each processor is configured with an input channel and has a resulting output channel. These are managed by the Manager, passing documents from one channel to the next.
Currently processing is performed by monitoring the `bundles.source_folder` for new files that have been added. These notifications come in via a registered webhook that is configured for the Google Drive folder. When a new file is detected, it is sent to the first Processor in the chain. This will use the passed in metadata `document.Document` and the `io.ReadCloser` from storage location. It will perform any necessayr processing then pass to the next Processor. The current processing chain that is configured is:

- TempStorage is used to copy the original PDF down from the storage location for staged processing.
- Mathpix is used to convert the PDF to a Markdown file.
- ChatGPT is used to take a Markdown file as input and clean it up for spelling, grammar, and correct Markdown syntax.
- Obsidian is a step that simply adds an Obsidian link at the end of the Markdown to include the original PDF attachment.
- BundleProcessor will read the bundle configuration from then config file and based on the `source_folder` copy the destination files to the configured destination.
