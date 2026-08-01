# EmlToMarkdown
Drag an EML file into the application and it will convert to simple markdown

# Format
EML file will be parsed and then Markdown output will be generated in the following format:

```
### EMAIL: [Subject Line Here]
* Date: YYYY-MM-DD HH:MM:SS UTC
* From: Jane Doe <jane@example.com>
* To: John Smith <john@example.com>

[Email body text / formatted thread]
```

# Attachments
Attachments are excluded from the result, so only the text body will be there. I may add that in the future.
