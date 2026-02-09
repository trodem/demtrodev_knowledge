# Git Functions Cheat Sheet

Generated from `plugins/functions/git.ps1`.

## g_add_all
Stage all changes.

```powershell
dm g_add_all
```

## g_add_commit
Stage all changes and commit.

```powershell
dm g_add_commit -Message "Update docs"
```

## g_branch_current
Show current branch name.

```powershell
dm g_branch_current
```

## g_branch_delete_local
Delete a local branch.

```powershell
dm g_branch_delete_local -Branch old/feature -Force
```

## g_branch_delete_remote
Delete a remote branch on origin.

```powershell
dm g_branch_delete_remote -Branch old/feature
```

## g_branch_list
List local and remote branches.

```powershell
dm g_branch_list
```

## g_branch_rename
Rename current branch.

```powershell
dm g_branch_rename -NewName feature/new-name
```

## g_cheatsheet
Show Git cheat sheet in terminal.

```powershell
dm g_cheatsheet
```

## g_checkout_new
Create and switch to a new branch.

```powershell
dm g_checkout_new -Branch feature/login
```

## g_cherry_pick
Cherry-pick one commit.

```powershell
dm g_cherry_pick -Ref abc1234
```

## g_commit
Commit staged changes.

```powershell
dm g_commit -Message "Fix parser"
```

## g_commit_amend
Amend latest commit with a new message.

```powershell
dm g_commit_amend -Message "Refine parser"
```

## g_commit_amend_noedit
Amend latest commit without changing message.

```powershell
dm g_commit_amend_noedit
```

## g_diff
Show unstaged and staged diff.

```powershell
dm g_diff
```

## g_diff_file
Show diff for a single file.

```powershell
dm g_diff_file -Path "README.md"
```

## g_fetch
Fetch updates from remotes.

```powershell
dm g_fetch
```

## g_grep
Search content across tracked files.

```powershell
dm g_grep -Pattern "TODO"
```

## g_log
Show recent commit history.

```powershell
dm g_log
```

## g_log_file
Show one-line log for a single file.

```powershell
dm g_log_file -Path "internal/app/app.go"
```

## g_log_graph_oneline
Show compact graph log.

```powershell
dm g_log_graph_oneline
```

## g_merge_abort
Abort an in-progress merge.

```powershell
dm g_merge_abort
```

## g_merge_main
Merge main/master into current branch.

```powershell
dm g_merge_main
```

## g_pull
Pull updates for current branch.

```powershell
dm g_pull
```

## g_pull_rebase
Pull with rebase strategy.

```powershell
dm g_pull_rebase
```

## g_push
Push current branch to remote.

```powershell
dm g_push
```

## g_push_force_with_lease
Push with force-with-lease.

```powershell
dm g_push_force_with_lease
```

## g_rebase_abort
Abort an in-progress rebase.

```powershell
dm g_rebase_abort
```

## g_rebase_continue
Continue an in-progress rebase.

```powershell
dm g_rebase_continue
```

## g_rebase_main
Rebase current branch on main/master.

```powershell
dm g_rebase_main
```

## g_remote_list
Show remote configuration.

```powershell
dm g_remote_list
```

## g_reset_mixed_head1
Move HEAD back by one commit (mixed).

```powershell
dm g_reset_mixed_head1
```

## g_reset_soft_head1
Move HEAD back by one commit (soft).

```powershell
dm g_reset_soft_head1
```

## g_restore_file
Restore one file from HEAD.

```powershell
dm g_restore_file -Path "README.md" -Confirm
```

## g_revert
Revert one commit.

```powershell
dm g_revert -Ref abc1234
```

## g_show
Show full details for one commit.

```powershell
dm g_show -Ref HEAD~1
```

## g_stash_apply
Apply stash without dropping it.

```powershell
dm g_stash_apply -Ref "stash@{1}"
```

## g_stash_clear
Clear all stash entries.

```powershell
dm g_stash_clear -Confirm
```

## g_stash_drop
Drop one stash entry.

```powershell
dm g_stash_drop -Ref "stash@{0}"
```

## g_stash_list
List stashes.

```powershell
dm g_stash_list
```

## g_stash_pop
Apply and drop top stash.

```powershell
dm g_stash_pop
```

## g_stash_push
Create a stash entry.

```powershell
dm g_stash_push -Message "WIP api"
```

## g_status
Show repository status.

```powershell
dm g_status
```

## g_submodule_update
Sync and init submodules.

```powershell
dm g_submodule_update
```

## g_switch
Switch to an existing branch.

```powershell
dm g_switch -Branch feature/login
```

## g_switch_main
Switch to main-like branch.

```powershell
dm g_switch_main
```

## g_tag_create
Create an annotated tag.

```powershell
dm g_tag_create -Name v1.2.0 -Message "Release v1.2.0"
```

## g_tag_list
Show tags list.

```powershell
dm g_tag_list
```

## g_tag_push
Push one tag to origin.

```powershell
dm g_tag_push -Name v1.2.0
```

## g_tag_push_all
Push all tags to origin.

```powershell
dm g_tag_push_all
```

## g_unstage_all
Unstage all files.

```powershell
dm g_unstage_all
```

## g_unstage_file
Unstage one file.

```powershell
dm g_unstage_file -Path "README.md"
```

## g_worktree_add
Add a git worktree for a branch.

```powershell
dm g_worktree_add -Path "../repo-hotfix" -Branch hotfix/urgent
```

## g_worktree_list
List git worktrees.

```powershell
dm g_worktree_list
```

## g_worktree_remove
Remove a git worktree.

```powershell
dm g_worktree_remove -Path "../repo-hotfix" -Force
```

