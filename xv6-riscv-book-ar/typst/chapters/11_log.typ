#import "../template/listings.typ": *

= Logging <CH:LOG>

One of the most interesting problems in file system design is crash
recovery. The problem arises because many file-system operations
involve multiple writes to the disk, and a crash after a subset of the
writes may leave the on-disk file system in an inconsistent state. For
example, suppose a crash occurs during file truncation.
Truncation involves writing (at least) two disk blocks,
one containing the i-node and one containing part of the free-block bitmap:
the i-node's length must be set to zero and the block
numbers in the i-node must be cleared,
and the bitmap bits for the file's blocks must
be set to zero (marking them free).
A crash after one of the writes, but before the kernel has
a chance to perform the second write, will leave the
on-disk file system in an incorrect state.
If only the bitmap write occurs, the result after a crash
and reboot will be an i-node with non-zero length which
refers to blocks that are marked free;
if only the i-node write occurs, the result will be
blocks that are not used by any file but are also not marked free.

The latter is relatively benign, but an inode that refers to a freed
block is likely to cause serious problems after a reboot.  After reboot, the
kernel might allocate that block to another file, and now we have two different
files pointing unintentionally to the same block.  If xv6 supported
multiple users, this situation could be a security problem, since the
old file's owner would be able to read and write blocks in the
new file, owned by a different user.

Xv6 solves the problem of crashes during file-system operations with a
simple form of logging. An xv6 system call does not directly write
the on-disk file system data structures. Instead, it places a
description of all the disk writes it wishes to make in a 
_log_ 
on the disk. Once the system call has logged all of its writes, it writes a
special 
_commit_
record to the disk indicating that the log contains
a complete operation. At that point the system call copies the writes
to the on-disk file system data structures. After those writes have
completed, the system call erases the log on disk.

If the system should crash and reboot, the file-system code recovers
from the crash as follows, before running any processes. If the log is
marked as containing a complete operation, then the recovery code
copies the writes to where they belong in the on-disk file system. If
the log is not marked as containing a complete operation, the recovery
code ignores the log.  The recovery code finishes by erasing
the log.

Why does xv6's log solve the problem of crashes during file system
operations? If the crash occurs before the operation commits, then the
log on disk will not be marked as complete, the recovery code will
ignore it, and the state of the disk will be as if the operation had
not even started. If the crash occurs after the operation commits,
then recovery will replay all of the operation's writes, perhaps
repeating them if the operation had started to write them to the
on-disk data structure. In either case, the log makes operations
atomic with respect to crashes: after recovery, either all of the
operation's writes appear on the disk, or none of them appear.

== Log design

The log resides at a known fixed location (see
Figure~@fig:fslayout), specified in the superblock.
It consists of a header block followed by a sequence
of updated block copies (``logged blocks'').
The header block contains an array of block
numbers, one for each of the logged blocks, and 
the count of log blocks.
The count in the header block on disk is either
zero, indicating that there is no transaction in the log,
or non-zero, indicating that the log contains a complete committed
transaction with the indicated number of logged blocks.
Xv6 writes the header
block when a transaction commits, but not before, and sets the
count to zero after copying the logged blocks to the file system.
Thus a crash midway through a transaction will result in a
count of zero in the log's header block; a crash after a commit
will result in a non-zero count.

Each system call's code indicates the start and end of the sequence of
writes that must be atomic with respect to crashes.
To allow concurrent execution of file-system operations
by different processes,
the logging system can accumulate the writes
of multiple system calls into one transaction.
Thus a single commit may involve the writes of multiple
complete system calls.
To avoid splitting a system call across transactions, the logging system
only commits when no file-system system calls are underway.

The idea of committing several transactions together is known as 
_group commit_.
Group commit reduces the number of disk operations
because it amortizes the fixed cost of a commit over multiple
operations.
Group commit also hands the disk system more concurrent writes
at the same time, perhaps allowing the disk to write
them all during a single disk rotation.
Xv6's virtio driver doesn't support this kind of
_batching_,
but xv6's file system design allows for it.

A subtle implication of group commit is
that it makes the effects of a write system call asynchronous:
that is, even after a write system call returns to the application, it is not guaranteed that
the blocks it modified are on disk.
For example, when a write transaction finishes, 
another transaction may have started and the write transaction will
commit together with the second transaction as one group, delaying the
effects of the write on disk until the second transaction commits.
If an application wants to wait until its modifications are made durable
(i.e., are on disk),
it can call the `sync` سطر `kernel/log.c:/sys_sync/` system call after a write system call;
`sync` waits until an in-progress group-commit has committed.

== Code: logging <s:code-logging>

#lstlisting[
[]
  begin_op();
  ...
  bp = bread(...);
  bp->data[...] = ...;
  log_write(bp);
  ...
  end_op();
]

`begin_op`
سطر `kernel/log.c:/^begin.op/`
waits until
the logging system is not currently committing, and until
there is enough unreserved log space to hold
the writes from this call.
`log.outstanding`
counts the number of system calls that have reserved log
space; the total reserved space is 
`log.outstanding`
times
`MAXOPBLOCKS`.
Incrementing
`log.outstanding`
both reserves space and prevents a commit
from occurring during this system call.
The code conservatively assumes that each system call might write up to
`MAXOPBLOCKS`
distinct blocks.

`log_write`
سطر `kernel/log.c:/^log.write/`
acts as a proxy for 
`bwrite`.
It records the block's block number in memory,
reserving it a slot in the log on disk,
and pins the buffer in the block cache
to prevent the block cache from evicting it.
The block must stay in the cache until committed:
until then, the cached copy is the only record
of the modification; it cannot be written to
its place on disk until after commit;
and other reads in the same transaction must
see the modifications.
`log_write`
notices when a block is written multiple times during a single
transaction, and allocates that block the same slot in the log.
This optimization is often called
_absorption_.
It is common that, for example, the disk block containing inodes
of several files is written several times within a transaction.  By absorbing
several disk writes into one, the file system can save log space and
can achieve better performance because only one copy of the disk block must be
written to disk.

`end_op`
سطر `kernel/log.c:/^end.op/`
first decrements the count of outstanding system calls.
If the count is now zero, it commits the current
transaction by calling
`commit().`
There are four stages in this process.
`write_log()`
سطر `kernel/log.c:/^write.log/`
copies each block modified in the transaction from the buffer
cache to its slot in the log on disk.
`write_head()`
سطر `kernel/log.c:/^write.head/`
writes the header block to disk: this is the
commit point, and a crash after the write will
result in recovery replaying the transaction's writes from the log.
`install_trans`
سطر `kernel/log.c:/^install_trans/`
reads each block from the log and writes it to the proper
place in the file system.
Finally
`end_op`
writes the log header with a count of zero;
this has to happen before the next transaction starts writing
logged blocks, so that a crash doesn't result in recovery
using one transaction's header with the subsequent transaction's
logged blocks.

A key assumption in the implementation of `end_op` is that
`write_head` updates the header block atomically: completely or
not at all.
Recall from Chapter~@CH:FS that a block in xv6 is two disk sectors, but the log header
fits in the first sector, and so there is no problem even if the disk
guarantees only that a single-sector update is atomic.

`recover_from_log`
سطر `kernel/log.c:/^recover_from_log/`
is called from 
`initlog`
سطر `kernel/log.c:/^initlog/`,
which is called from `fsinit`سطر `kernel/fs.c:/^fsinit/` during boot before the first user process runs
سطر `kernel/proc.c:/fsinit/`.
It reads the log header, and mimics the actions of
`end_op`
if the header indicates that the log contains a committed transaction.

#lstlisting[
[]
      begin_op();
      ilock(f->ip);
      r = writei(f->ip, ...);
      iunlock(f->ip);
      end_op();
]

Xv6's logging system is inefficient.
A commit cannot occur concurrently with file-system system calls.
The system logs entire blocks, even if
only a few bytes in a block are changed. It performs synchronous
log writes, a block at a time, each of which is likely to require an
entire disk rotation time. Real logging systems address all of these
problems.
-
Logging is not the only way to provide crash recovery. Early file systems
used a scavenger during reboot (for example, the UNIX
`fsck`
program) to examine every file and directory and the block and inode
free lists, looking for and resolving inconsistencies. Scavenging can take
hours for large file systems, and there are situations where it is not
possible to resolve inconsistencies in a way that causes the original
system calls to be atomic. Recovery
from a log is much faster and causes system calls to be atomic
in the face of crashes.

\begin{enumerate}

\item To understand how the log is used, insert the following line
    of code before the `log_write` statement in the function
    `writei()` in `kernel/fs.c`:

\begin{verbatim}
printf("log write %d: %d %d %d\n", addr, bp->blockno, off, n);
\end{verbatim}

Run make clean and then make qemu, and at the xv6 shell prompt run the following command:
\begin{verbatim}
$ echo hi > f
\end{verbatim}

You should see output similiar to this:
\begin{verbatim}
log write 47: 47 368 16
log write 936: 936 0 2
log write 936: 936 2 1
\end{verbatim}

The output indicates that `log_write` is called three times
for this shell command. Why is `log_write` called 3 times? What does
each `log_write` write? Does the first write above correspond to
updating the root directory and the other two writes correspond to
writing file f?  Figure~@fig:fslayout might be helpful and feel
free to add more `printf` statements to help you answering
the question.

\end{enumerate}
