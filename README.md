# Scriptoria

## Purpose

Scriptoria is a service that is deployed and will monitor a source store either
located either locally or in the "cloud". It will look for PDF files that have
changed. On detection of a new file or a file that has changed it will download
the files and perform OCR (Optical Character Recognition) on them and convert
any text to a Markdown file of the same name. It will then upload the file to
a configured destination store.

