# Jira sync

Sync GitLab issues to Jira and marked the synced GitLab issues with a "ticket-created" label.

[![Github license](https://img.shields.io/github/license/gabor-boros/jira-sync)](https://github.com/gabor-boros/jira-sync/)

## Installation

To install `jira-sync`, use one of the [release artifacts](https://github.com/gabor-boros/jira-sync/releases) or simply run `go install https://github.com/gabor-boros/jira-sync`

You must create a new configuration file `$HOME/.jira-sync.toml` with the following content:

```toml
gitlab_token = "<GitLab user access token>"
jira_url = "<Jira server URL>"
jira_username = "<Jira username>"
jira_password = "<Jira password>"
```

## Usage

```plaintext
Sync GitLab issues to Jira and marked the synced GitLab issues with a
"ticket-created" label.

Usage:
  jira-sync [flags]

Flags:
      --config string           config file (default is $HOME/.jira-sync.yaml)
  -i, --gitlab-issue ints       GitLab issue IDs
      --gitlab-project string   GitLab project name
  -h, --help                    help for jira-sync
      --jira-account string     Jira account key (ex: ABC)
      --jira-epic string        Jira epic key (ex: SE-1234)
      --jira-project string     Jira project key (ex: SE)
      --link-as-description     use the GitLab issue link as the description of the jira ticket
      --version                 show command version
```

## Development

To install everything you need for development, run the following:

```shell
$ git clone git@github.com:gabor-boros/jira-sync.git
$ cd jira-sync
$ make prerequisites
$ make deps
```

## Contributors

- [gabor-boros](https://github.com/gabor-boros)
