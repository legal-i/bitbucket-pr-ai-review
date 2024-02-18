# Bitbucket PR Review Tool 

This is a Go application that reviews Pull Requests (PRs) from Bitbucket with the Azure OpenAI API.
 
To get started, you'll need to set up the necessary environment variables as described below.

## Prerequisites

Before you can use this tool, make sure you have the following prerequisites:

- Bitbucket account with the PRs you want to review.
- Azure OpenAI API Key and Base URL for code analysis.
- Bitbucket Username and App Password for authentication.

## Environment Variables

To ensure the tool works seamlessly, you need to set the following environment variables:

- **OPENAI_API_KEY:** Your API key for OpenAI.
- **BITBUCKET_USERNAME:** Your Bitbucket username, which will be used for authentication.
- **BITBUCKET_APP_PASSWORD:** An App Password generated from your Bitbucket account. 
This password should have appropriate access rights to the repositories you want to review PRs.

You can set these environment variables in your system or create a `.env` file in the same directory as this tool and populate it with the necessary values. 
Here's an example of how the `.env` file should look:

```dotenv
OPENAI_API_KEY=your_azure_openai_api_key
BITBUCKET_USERNAME=your_bitbucket_username
BITBUCKET_APP_PASSWORD=your_bitbucket_app_password
```

## Installation

1. Clone this repository to your local machine:

   ```bash
   git clone https://github.com/yourusername/bitbucket-pr-review-tool-go.git
   cd bitbucket-pr-review-tool-go
   ```

2. Build the Go application:

   ```bash
   make build
   ```

3. Ensure your environment variables are set correctly, as explained in the "Environment Variables" section.

## Usage

Once you have set up the environment variables and built the Go application, you can use the tool to review Bitbucket PRs with the Azure OpenAI API. Here's how to use it:

Run the application with the desired Bitbucket PR URL:

   ```bash
   ./bitbucket-pr-review-tool-go <Pull Request ID>
   ```

The tool will fetch the diff of the PR with the Bitbucket Rest API, analyze the PR using the Azure OpenAI API and add a comment to the PR


## Support and Feedback

If you encounter any issues or have suggestions for improvements, please open an issue on this GitHub repository. We welcome your feedback to make this Go application better.

Happy reviewing! 🚀
