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

Running all tests here in the https://github.com/apache/bookkeeper repo

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

when it gets stuck I see one or two ls-tree commands

```
 PID  PPID  C    SZ   RSS PSR STIME TTY          TIME CMD
4149  3874  0 18113  9688   0 17:40 pts/1    00:00:00 git ls-tree -r c9dc301feb48ca170c3d6205a36fca63a4950c5a
4162  3874  0 18113  8820   1 17:40 pts/1    00:00:00 git ls-tree -r 33ea58027b0a3ba160f7ac19d20568709f453f4d
```

this is how the goroutine stack traces look like then

```
985  syscall             syscall.Syscall6(0xf7, 0x1, 0x1035, 0xc0018d3d30, 0x1000004, 0x0, 0x0, 0xc0020774e0, 0x14f, 0x1)
     syscall, 10 minutes
       /usr/local/go/src/syscall/asm_linux_amd64.s:43 +0x5
     os.(*Process).blockUntilWaitable(0xc0002e8120, 0x4, 0x4, 0x203000)
       /usr/local/go/src/os/wait_waitid.go:32 +0x9e
     os.(*Process).wait(0xc0002e8120, 0x8, 0x7feaf0, 0x7feaf8)
       /usr/local/go/src/os/exec_unix.go:22 +0x39
     os.(*Process).Wait(...)
       /usr/local/go/src/os/exec.go:129
     os/exec.(*Cmd).Wait(0xc001a36420, 0x0, 0x0)
       /usr/local/go/src/os/exec/exec.go:507 +0x65
     os/exec.(*Cmd).Run(0xc001a36420, 0xc000fbb770, 0xc001a36420)
       /usr/local/go/src/os/exec/exec.go:341 +0x5f
     os/exec.(*Cmd).CombinedOutput(0xc001a36420, 0x3, 0xc000595788, 0x3, 0x3, 0xc001a36420)
       /usr/local/go/src/os/exec/exec.go:567 +0x91
     main.putBlobsInChannel(0xc000269544, 0x28, 0xc00011ca80)
       /home/ivo/code/talisman-experiments/scanner-profiling/scanner.go:159 +0xe9
     created by main.getBlobsInCommit
       /home/ivo/code/talisman-experiments/scanner-profiling/scanner.go:149 +0xd1
3097 syscall             syscall.Syscall6(0x3d, 0x1042, 0xc0009f4b14, 0x0, 0x0, 0x0, 0x0, 0xc0009f4ac8, 0x46d7e5, 0xc000ab8180)
     syscall, 10 minutes
       /usr/local/go/src/syscall/asm_linux_amd64.s:43 +0x5
     syscall.wait4(0x1042, 0xc0009f4b14, 0x0, 0x0, 0x0, 0xffffffffffffffff, 0x0)
       /usr/local/go/src/syscall/zsyscall_linux_amd64.go:168 +0x76
     syscall.Wait4(0x1042, 0xc0009f4b9c, 0x0, 0x0, 0x853ec0, 0xa069a8, 0x38)
       /usr/local/go/src/syscall/syscall_linux.go:368 +0x51
     syscall.forkExec(0xc000aaa090, 0xc, 0xc000901f40, 0x4, 0x4, 0xc0009f4ce0, 0x37, 0x6890bc1200010400, 0xc00212d000)
       /usr/local/go/src/syscall/exec_unix.go:237 +0x558
     syscall.StartProcess(...)
       /usr/local/go/src/syscall/exec_unix.go:263
     os.startProcess(0xc000aaa090, 0xc, 0xc000901f40, 0x4, 0x4, 0xc0009f4e70, 0xc002135880, 0x37, 0x37)
       /usr/local/go/src/os/exec_posix.go:53 +0x29b
     os.StartProcess(0xc000aaa090, 0xc, 0xc000901f40, 0x4, 0x4, 0xc0009f4e70, 0x37, 0x1ed, 0x203000)
       /usr/local/go/src/os/exec.go:106 +0x7c
     os/exec.(*Cmd).Start(0xc000a10000, 0x1, 0xc0020ea3f0)
       /usr/local/go/src/os/exec/exec.go:422 +0x525
     os/exec.(*Cmd).Run(0xc000a10000, 0xc0020ea3f0, 0xc000a10000)
       /usr/local/go/src/os/exec/exec.go:338 +0x2b
     os/exec.(*Cmd).CombinedOutput(0xc000a10000, 0x3, 0xc000ab7788, 0x3, 0x3, 0xc000a10000)
       /usr/local/go/src/os/exec/exec.go:567 +0x91
     main.putBlobsInChannel(0xc00027e784, 0x28, 0xc00011ca80)
       /home/ivo/code/talisman-experiments/scanner-profiling/scanner.go:159 +0xe9
     created by main.getBlobsInCommit
       /home/ivo/code/talisman-experiments/scanner-profiling/scanner.go:149 +0xd1
```

one is doing a blockUntilWaitable and the other seems to be doing the syscall
that executes the `git ls-tree`

Note this output is cut short to 99 characters using the option -s

```sh
strace -s 99 -ffp 4149
strace: Process 4149 attached
write(1, "AuditorElector.java\n100644 blob 9830c592904cf4848d4068b64e562b78e815b5dc\tbookkeeper-server/src/main"..., 4096
```

this to

```sh
strace -s 1000000000 -p 4162
write(1, "e/hedwig/client/benchmark/BenchmarkPublisher.java\n100644 blob 0f8cb7f381c7407e63601cf737cddab530d20123\thedwig-client/src/main/java/org/apache/hedwig/client/benchmark/BenchmarkSubscriber.java\n100644 blob 3efe22da20938a875dee575044d9c5e4e9d234b0\thedwig-client/src/main/java/org/apache/hedwig/client/benchmark/BenchmarkUtils.java\n100644 blob e7b15f26a2ffef6e44573adf95a011e296007fd8\thedwig-client/src/main/java/org/apache/hedwig/client/benchmark/BenchmarkWorker.java\n100644 blob cc5e93778a041724de8a050276fcc3497f14c21b\thedwig-client/src/main/java/org/apache/hedwig/client/benchmark/HedwigBenchmark.java\n100644 blob 21ce9d3b34c9bec19eee58fba6001bedb63c2f46\thedwig-client/src/main/java/org/apache/hedwig/client/conf/ClientConfiguration.java\n100644 blob 346d74b34b1a728f38b0a74e036fc88b1c0e8474\thedwig-client/src/main/java/org/apache/hedwig/client/data/MessageConsumeData.java\n100644 blob 63547a0fdafff58646fe83f713c16d9741aa0abd\thedwig-client/src/main/java/org/apache/hedwig/client/data/PubSubData.java\n100644 blob 064cec12d379684adec3a4f33a46f22625919783\thedwig-client/src/main/java/org/apache/hedwig/client/data/TopicSubscriber.java\n100644 blob 5f468e6d3f5b05408946f3485861e8004d13f030\thedwig-client/src/main/java/org/apache/hedwig/client/exceptions/AlreadyStartDeliveryException.java\n100644 blob 3e543569f09f1dab37b23542115faeb85c088e85\thedwig-client/src/main/java/org/apache/hedwig/client/exceptions/InvalidSubscriberIdException.java\n100644 blob 22b44b16f649b0efd93b9530164ae9aad9b962e5\thedwig-client/src/main/java/org/apache/hedwig/client/exceptions/NoResponseHandlerException.java\n100644 blob c9aeb385307340e75c03e24195d333ef0fbc5933\thedwig-client/src/main/java/org/apache/hedwig/client/exceptions/ResubscribeException.java\n100644 blob da6d4e7d39ee0a1359a9f2dcb364697e3ae25384\thedwig-client/src/main/java/org/apache/hedwig/client/exceptions/ServerRedirectLoopException.java\n100644 blob 4a3c99f0f42beea2858fc203a824a1d93a2a3885\thedwig-client/src/main/java/org/apache/hedwig/client/exceptions/TooManyServerRedirectsException.java\n100644 blob bb2c0bb658b8bdef6f7b535df671a857a0b0df06\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/AbstractResponseHandler.java\n100644 blob 102dfb509a450fef90116e97982960b1f7dda258\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/CloseSubscriptionResponseHandler.java\n100644 blob 436c14f85b5e65be42196f14d5160ecc4db652ee\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/MessageConsumeCallback.java\n100644 blob dacaa7aa715e6099810d58d3831a2a9376d588b0\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/PubSubCallback.java\n100644 blob fc6a0251074488ef169090531dd8c7336e12681d\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/PublishResponseHandler.java\n100644 blob e2c685f91d687e8b709653af50b6fe3dcefa0231\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/SubscribeResponseHandler.java\n100644 blob 3ddd5390553150162e9482d6e2125998cb12fde2\thedwig-client/src/main/java/org/apache/hedwig/client/handlers/UnsubscribeResponseHandler.java\n100644 blob 0c676a13c909580f1aa85105fa54d1eb6469e273\thedwig-client/src/main/java/org/apache/hedwig/client/netty/CleanupChannelMap.java\n100644 blob 94e0a808e7858020c4d0f3692126b7590bc169bb\thedwig-client/src/main/java/org/apache/hedwig/client/netty/FilterableMessageHandler.java\n100644 blob 340cec57553513c96524c12f7f2826648107581e\thedwig-client/src/main/java/org/apache/hedwig/client/netty/HChannel.java\n100644 blob 6fae6bb2588d6d6b666df72793c3628c16fba38e\thedwig-client/src/main/java/org/apache/hedwig/client/netty/HChannelManager.java\n100644 blob 8ae0e8207e171f4d8b79ca9e605f573709884ca0\thedwig-client/src/main/java/org/apache/hedwig/client/netty/HedwigClientImpl.java\n100644 blob 5611bdd0c6e5f6871ec1fd6c751f6b16761aa2e6\thedwig-client/src/main/java/org/apache/hedwig/client/netty/HedwigPublisher.java\n100644 blob 7d2453aa29d477dd823ec0bbeb5a183e3efce531\thedwig-client/src/main/java/org/apache/hedwig/client/netty/HedwigSubscriber.java\n100644 blob 1d4f95555ac34c9ddff68e913c0b865b09de581c\thedwig-client/src/main/java/org/apache/hedwig/client/netty/NetUtil", 4096
```
