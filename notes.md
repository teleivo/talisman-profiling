# Scanner

## Design

- why is exec used to executed git commands instead of

```go
byteArray := repo.executeRepoCommand("git", "ls-tree", branchName, "--name-only", "-r")
```

like for example in the detector/chain.go?

## Performance

Regarding the OOM issue

- lazily read blob content into memory, just before checking patterns. Currently scanner does `git cat-file -p` on every reachable blob in the history when doing a --scan
- scanner also reads blob contents into memory of files that are ignored in
  the .talismanrc
- reuse git sha as checksum to avoid reading blob contents of files that are
  ignored

### Hanging issues

`talisman --scan` sometimes hangs on a `git ls-tree -r`. I can see that it is
happening when I see one or more `git ls-tree -r` subprocesses of my
`talisman --scan` in htop.

Is the number of subprocesses I see matching the number of commits in a repo? I
should certainly see # commits goroutines. I would not expect to see goroutines
as subprocesses in something like htop though. So what are these subprocesses?
This is also happening when my machine is pretty idle. So what is happening?

Could it be particular trees that are deep? Or contain many blobs in total?
A quick check with git-sizer --verbose showing me the largest tree did not work
out. However, this just shows me the largest tree.

Do they relate in any way?

```sh
git rev-list --all --count
pstree -g -a 22175
ps --ppid 22175 # interesting that this does only show the one or two hanging git ls-tree commands
```

So what are these processes showing up?
How can I figure out if the `git ls-tree` ones are actually running or got the
chance to do some work? Are they stuck? Where they never really started?

Is the sequential scan also getting stuck or just the goroutine version???? ;)

When doing the sequential scan I can see `git ls-tree` subprocesses popping up
and shutting down in htop :) Does not seem to get stuck.
