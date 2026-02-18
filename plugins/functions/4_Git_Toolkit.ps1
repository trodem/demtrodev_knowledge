# =============================================================================
# DM GIT TOOLKIT – Local Git Operational Layer
# Production-safe Git helpers for local development environments
# Non-destructive defaults, deterministic behavior, no admin requirements
# Entry point: git_*
#
# FUNCTIONS
#   git_status
#   git_branch_current
#   git_branch_list
#   git_fetch
#   git_pull
#   git_pull_rebase
#   git_push
#   git_push_force_with_lease
#   git_add_all
#   git_commit
#   git_add_commit
#   git_commit_amend
#   git_commit_amend_noedit
#   git_log
#   git_log_graph_oneline
#   git_diff
#   git_diff_file
#   git_log_file
#   git_grep
#   git_show
#   git_remote_list
#   git_tag_list
#   git_tag_create
#   git_tag_push
#   git_tag_push_all
#   git_switch
#   git_checkout_new
#   git_branch_delete_local
#   git_branch_delete_remote
#   git_rebase_continue
#   git_rebase_abort
#   git_merge_abort
# =============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

<#
.SYNOPSIS
Ensure current directory is inside a Git repository.
.DESCRIPTION
Throws an error when Git metadata is not available for the current path.
.EXAMPLE
_assert_git_repo
#>
function _assert_git_repo {
    _assert_command_available -Name git
    git rev-parse --is-inside-work-tree 1>$null 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Current directory is not a Git repository."
    }
}

function git_status { _assert_git_repo; git status }
function git_branch_current { _assert_git_repo; git branch --show-current }
function git_branch_list { _assert_git_repo; git branch -a }
function git_fetch { _assert_git_repo; git fetch --all --prune }
function git_pull { _assert_git_repo; git pull }
function git_pull_rebase { _assert_git_repo; git pull --rebase }
function git_push { _assert_git_repo; git push }
function git_push_force_with_lease { _assert_git_repo; git push --force-with-lease }
function git_add_all { _assert_git_repo; git add . }

function git_commit {
    param([Parameter(Mandatory=$true)][string]$Message)
    _assert_git_repo
    git commit -m "$Message"
}

function git_add_commit {
    param([Parameter(Mandatory=$true)][string]$Message)
    _assert_git_repo
    git add .
    git commit -m "$Message"
}

function git_commit_amend {
    param([Parameter(Mandatory=$true)][string]$Message)
    _assert_git_repo
    git commit --amend -m "$Message"
}

function git_commit_amend_noedit { _assert_git_repo; git commit --amend --no-edit }
function git_log { _assert_git_repo; git log --decorate --stat -n 20 }
function git_log_graph_oneline { _assert_git_repo; git log --oneline --graph --decorate --all }
function git_diff { _assert_git_repo; git diff; git diff --cached }

function git_diff_file {
    param([Parameter(Mandatory=$true)][string]$Path)
    _assert_git_repo
    git diff -- "$Path"
    git diff --cached -- "$Path"
}

function git_log_file {
    param([Parameter(Mandatory=$true)][string]$Path)
    _assert_git_repo
    git log --oneline -- "$Path"
}

function git_grep {
    param([Parameter(Mandatory=$true)][string]$Pattern)
    _assert_git_repo
    git grep -- "$Pattern"
}

function git_show {
    param([string]$Ref="HEAD")
    _assert_git_repo
    git show "$Ref"
}

function git_remote_list { _assert_git_repo; git remote -v }
function git_tag_list { _assert_git_repo; git tag --list }

function git_tag_create {
    param(
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$Message
    )
    _assert_git_repo
    git tag -a "$Name" -m "$Message"
}

function git_tag_push {
    param([Parameter(Mandatory=$true)][string]$Name)
    _assert_git_repo
    git push origin "$Name"
}

function git_tag_push_all { _assert_git_repo; git push --tags }

function git_switch {
    param([Parameter(Mandatory=$true)][string]$Branch)
    _assert_git_repo
    git switch "$Branch"
}

function git_checkout_new {
    param([Parameter(Mandatory=$true)][string]$Branch)
    _assert_git_repo
    git checkout -b "$Branch"
}

function git_branch_delete_local {
    param(
        [Parameter(Mandatory=$true)][string]$Branch,
        [switch]$Force
    )
    _assert_git_repo
    if ($Force) { git branch -D "$Branch" } else { git branch -d "$Branch" }
}

function git_branch_delete_remote {
    param([Parameter(Mandatory=$true)][string]$Branch)
    _assert_git_repo
    git push origin --delete "$Branch"
}

function git_rebase_continue { _assert_git_repo; git rebase --continue }
function git_rebase_abort { _assert_git_repo; git rebase --abort }
function git_merge_abort { _assert_git_repo; git merge --abort }
