#import "../template/listings.typ": *

= File system <CH:FS>

The purpose of a file system is to organize and store data. File systems
typically support sharing of data among users and applications, as well as
_persistence_
so that data is still available after a reboot.

The xv6 file system provides Unix-like files, directories, and pathnames
(see Chapter~@CH:UNIX), and stores its data on a disk for
persistence. The file system addresses
several challenges:
\begin{itemize}
  
\item The file system needs on-disk data structures to represent the tree
of named directories and files, to record the identities of the
blocks that hold each file's content, and to record which areas
of the disk are free.
\item Different processes may operate on the file system at the same time,
so the file-system code must coordinate to maintain invariants.
\item Accessing a disk is orders of magnitude slower than accessing
memory, so the file system must maintain an in-memory cache of
popular blocks.
\item The file system must support
_crash recovery_.
That is, if a crash (e.g., power failure) occurs, the file system must
still work correctly after a restart. The risk is that a crash might
interrupt a sequence of updates and leave inconsistent on-disk data
structures (e.g., a block that is both used in a file and marked free).

\end{itemize}

The rest of this chapter explains how xv6 addresses the first three
challenges,
while Chapter~@CH:LOG
focuses on crash recovery.

The xv6 file system implementation is
organized in seven layers, shown in 
Figure~@fig:fslayer.
The disk layer reads and writes blocks on a virtio hard drive;
virtio is an emulated disk provided by `qemu`.
The buffer cache layer caches disk blocks and synchronizes access to them,
making sure that only one kernel process at a time can modify the
data stored in any particular block.  The logging layer allows higher
layers to wrap updates to several blocks in a
_transaction_,
and ensures that the blocks are updated atomically in the
face of crashes (i.e., all of them are updated or none).
The inode layer provides individual files, each represented as an
_inode_
with a unique i-number
and some blocks holding the file's data.  The directory
layer implements each directory as a special kind of
inode whose content is a sequence of directory entries, each of which contains a
file's name and i-number.
The pathname layer provides
hierarchical path names like
`/usr/rtm/xv6/fs.c`,
and resolves them with recursive lookup.
The file descriptor layer abstracts many Unix resources (e.g., pipes, devices,
files, etc.) using the file system interface, simplifying the lives of
application programmers.

\begin{figure}[t]
\center
\input{fig/fslayer.tex}
\caption{Layers of the xv6 file system.}
<fig:fslayer>
\end{figure}

Disk hardware traditionally presents the data on the
disk as a numbered sequence of 512-byte 
_sectors_\index{sector}:
sector 0 is the first 512 bytes, sector 1 is the next, and so on.
The disk hardware supports reads and writes only in whole multiples
of sectors. Thus, for example, if the operating system needs to change
a single byte on the disk, it must read the whole surrounding sector
into memory, update the one byte, and then write the whole sector
back to the disk. For this reason, file systems allocate disk space
for files in units of one or more sectors. The allocation granularity
that a file system uses is called the block \index{block} size.
Most file systems use multi-sector
blocks for better efficiency;
xv6 uses a block size of two sectors (determined by
`BSIZE` in `fs.h`).

Xv6 holds copies of blocks that it has read into memory
in objects of type
`struct buf`
سطر `kernel/buf.h:/^struct.buf/`.
The
data stored in this structure is sometimes 
not the same as the data on the disk: it might have
not yet been read in from disk (the disk is working on it but hasn't returned
the block's content yet), or it might have been updated by software
but not yet written to the disk.

\begin{figure}[t]
\center
\input{fig/fslayout.tex}
\caption{Structure of the xv6 file system. }
<fig:fslayout>
\end{figure}

The file system must have a plan for where it stores inodes and
content blocks on the disk.
To do so, xv6 divides the disk into several
sections, as 
Figure~@fig:fslayout shows.
The file system does not use
block 0 (block 0 often holds boot code, though in xv6 it's
merely unused).
Block 1 is called the 
_superblock_; 
it contains metadata about the file system (the file system size in blocks, the
number of data blocks, the number of inodes, and the number of blocks in the
log).  Blocks starting at 2 hold the log.  After the log are the inodes, with multiple inodes per block.  After
those come bitmap blocks tracking which data blocks are in use.
The remaining blocks are data blocks; each is either marked
free in the bitmap block, or holds content for a file or directory.

An xv6 file system is initially created by a program outside
of xv6 called 
`mkfs`.
`mkfs` writes the superblock, initializes an inode for
the (empty) root directory, and marks all i-nodes and data
blocks as free.

At this point please read the files `kernel/buf.h`
`kernel/buf.h`, `kernel/fs.h` `kernel/fs.h`,
`kernel/fs.c` `kernel/fs.c`,
`kernel/sysfile.c` `kernel/sysfile.c`, and
`kernel/file.c` `kernel/file.c`.

== Buffer cache layer <s:bcache>

The buffer cache has two jobs: (1) synchronize access to disk blocks to ensure
that only one copy of a block is in memory and that only one kernel thread at a time
uses that copy; (2) cache popular blocks so that they don't need to be re-read from
the slow disk. The code is in
`bio.c`.

The main interface exported by the buffer cache consists of
`bread`
and
`bwrite`;
the former obtains a
_buf_
containing a copy of a block which can be read or modified in memory, and the
latter writes a modified buffer to the appropriate block on the disk.
A kernel thread must release a buffer by calling
`brelse`
when it is done with it.
The buffer cache uses a per-buffer sleep-lock to ensure
that only one thread at a time uses each buffer
(and thus each disk block);
`bread`
returns a locked buffer, and
`brelse`
releases the lock.

The buffer cache has a fixed number of buffers to hold disk blocks,
which means that if the file system asks for a block that is not
already in the cache, the buffer cache must recycle a buffer currently
holding some other block. The buffer cache recycles the
least recently used buffer for the new block. The assumption is that
the least recently used buffer is the one least likely to be used
again soon.

There are two interactions between the buffer cache layer and the
logging layer: 1) to implement transactions, file-system layers above the logging layer update a disk
block by calling `log_write` سطر `kernel/log.c:/log_write/`, which is a proxy for
`bwrite` and only the logging layer
calls `bwrite`;  2)
the buffer cache cannot recycle a buffer that is in use by a
transaction until the logging layer commits the transaction (see
Section~@s:code-logging).
To avoid recycling too early, the logging layer uses
the buffer-cache functions `bpin` and `bunpin` to
pin and unpin a buffer in the buffer cache.

At this point please read the files `kernel/bio.c`
`kernel/bio.c` and
`kernel/log.c` `kernel/log.c`.

The buffer cache is a doubly-linked list of buffers.
The function
`binit`,
called by
`main`
سطر `kernel/main.c:/binit/`,
initializes the list with the
`NBUF`
buffers in the static array
`buf`
\linerefs{kernel/bio.c:/Create.linked.list/,/^..}/}.
All other access to the buffer cache refer to the linked list via
`bcache.head`,
not the
`buf`
array.

A buffer has two state fields associated with it.
The field
`valid`
indicates that the buffer contains a copy of the block.
The field `disk`
indicates that the buffer content has been handed to
the disk, which may change the buffer (e.g., write
data from the disk into `data`).

`bread`
سطر `kernel/bio.c:/^bread/`
calls
`bget`
to get a buffer for the given block
سطر `kernel/bio.c:/b.=.bget/`.
If the buffer needs to be read from disk,
`bread`
calls
`virtio_disk_rw`
to do that before returning the buffer.

`bget`
سطر `kernel/bio.c:/^bget/`
scans the buffer list for a buffer with the given device and block numbers
\linerefs{kernel/bio.c:/Is.the.block.already/,/^..}/}.
If there is such a buffer,
`bget`
acquires the sleep-lock for the buffer.
`bget`
then returns the locked buffer.

If there is no cached buffer for the given block,
`bget`
must make one, possibly reusing a buffer that held
a different block.
It scans the buffer list a second time, looking for a buffer
that is not in use (`b->refcnt = 0`);
any such buffer can be used.
`bget`
edits the buffer metadata to record the new device and block number
and acquires its sleep-lock.
Note that the assignment
`b->valid = 0`
ensures that
`bread`
will read the block data from disk
rather than incorrectly using the buffer's previous contents.

It is important that there is at most one cached buffer per
disk block, to ensure that readers see writes, and because the
file system uses locks on buffers for synchronization.
`bget`
ensures this invariant by holding the
`bcache.lock`
continuously from the first loop's check of whether the
block is cached through the second loop's declaration that
the block is now cached (by setting
`dev`,
`blockno`,
and
`refcnt`).
This causes the check for a block's presence and (if not
present) the designation of a buffer to hold the block to
be atomic.

It is safe for
`bget`
to acquire the buffer's sleep-lock outside of the 
`bcache.lock`
critical section,
since the non-zero
`b->refcnt`
prevents the buffer from being re-used for a different disk block.
The sleep-lock protects reads
and writes of the block's buffered content, while the
`bcache.lock`
protects information about which blocks are cached.

If all the buffers are busy, then too many processes are
simultaneously executing file system calls;
`bget`
panics.
A more graceful response might be to sleep until a buffer became free,
though there would then be a possibility of deadlock.

Once
`bread`
has read the disk (if needed) and returned the
buffer to its caller, the caller has
exclusive use of the buffer and can read or write the data bytes.
If the caller does modify the buffer, it must call
`bwrite`
to write the changed data to disk before releasing the buffer.
`bwrite`
سطر `kernel/bio.c:/^bwrite/`
calls
`virtio_disk_rw`
to talk to the disk hardware.

== Code: Block allocator

File and directory content is stored in disk blocks,
which must be allocated from a free pool.
Xv6's block allocator
maintains a free bitmap on disk, with one bit per block. 
A zero bit indicates that the corresponding block is free;
a one bit indicates that it is in use.
When it creates a new file system,
`mkfs`
sets the bits corresponding to the boot sector, superblock, log blocks, inode
blocks, and bitmap blocks.

The block allocator provides two functions:
`balloc`
allocates a new disk block, and
`bfree`
frees a block.
The loop in
`balloc`
at
سطر `kernel/fs.c:/^..for..b.=.0/`
considers every block, starting at block 0 up to 
`sb.size`,
the number of blocks in the file system.
It looks for a block whose bitmap bit is zero,
indicating that it is free.
If
`balloc`
finds such a block, it updates the bitmap 
and returns the block.
For efficiency, the loop is split into two 
pieces.
The outer loop reads each block of bitmap bits.
The inner loop checks all 
Bits-Per-Block (`BPB`)
bits in a single bitmap block.
The race that might occur if two processes try to allocate
a block at the same time is prevented by the fact that
the buffer cache only lets one process use any one bitmap block at a
time (see Section~@s:bcache).

`bfree`
سطر `kernel/fs.c:/^bfree/`
finds the right bitmap block and clears the right bit.
Again the exclusive use implied by
`bread`
and
`brelse`
avoids the need for explicit locking.

== Inode layer

The term 
_inode_ 
can have one of two related meanings.
It might refer to the on-disk data structure containing
a file's size and list of data block numbers.
Or ``inode'' might refer to an in-memory inode, which contains
a copy of the on-disk inode as well as extra information needed
within the kernel.

The on-disk inodes
are packed into a contiguous area
of disk called the inode blocks.
Every inode is the same size, so it is easy, given a
number n, to find the nth inode on the disk.
In fact, this number n, called the inode number or i-number,
is how inodes are identified in the implementation.

The on-disk inode is defined by a
`struct dinode`
سطر `kernel/fs.h:/^struct.dinode/`.
The 
`type`
field distinguishes between files, directories, and special
files (devices).
A type of zero indicates that an on-disk inode is free.
The
`nlink`
field counts the number of directory entries that
refer to this inode, in order to recognize when the
on-disk inode and its data blocks should be freed.
The
`size`
field records the number of bytes of content in the file.
The
`addrs`
array records the block numbers of the disk blocks holding
the file's content.

The kernel keeps the set of active inodes in memory
in a table called `itable`;
`struct inode`
سطر `kernel/file.h:/^struct.inode/`
is the in-memory copy of a 
`struct`
`dinode`
on disk.
The kernel stores an inode in memory only if there are
C pointers referring to that inode. The
`ref`
field counts the number of C pointers referring to the
in-memory inode, and the kernel discards the inode from
memory if the reference count drops to zero.
The
`iget`
and
`iput`
functions acquire and release pointers to an inode,
modifying the reference count.
Pointers to an inode can come from file descriptors,
current working directories, and transient kernel code
such as
`kexec`.

There are four lock or lock-like mechanisms in xv6's
inode code.
`itable.lock`
protects the invariant that an inode is present in the inode table
at most once, and the invariant that an in-memory inode's
`ref`
field counts the number of in-memory pointers to the inode.
Each in-memory inode has a
`lock`
field containing a
sleep-lock, which ensures exclusive access to the
inode's fields (such as file length) as well as to the
inode's file or directory content blocks.
An inode's
`ref`,
if it is greater than zero, causes the system to maintain
the inode in the table, and not re-use the table entry for
a different inode.
Finally, each inode contains a
`nlink`
field (on disk and copied in memory if in memory) that
counts the number of directory entries that refer to a file;
xv6 won't free an inode if its link count is greater than zero.

A
`struct`
`inode`
pointer returned by
`iget()`
is guaranteed to be valid until the corresponding call to
`iput()`;
the inode won't be deleted, and the memory referred to
by the pointer won't be re-used for a different inode.
`iget()`
provides non-exclusive access to an inode, so that
there can be many pointers to the same inode.
Many parts of the file-system code depend on this behavior of
`iget()`,
both to hold long-term references to inodes (as open files
and current directories) and to prevent races while avoiding
deadlock in code that manipulates multiple inodes (such as
pathname lookup).

The
`struct`
`inode`
that 
`iget`
returns may not have any useful content.
In order to ensure it holds a copy of the on-disk
inode, code must call
`ilock`.
This locks the inode (so that no other process can
`ilock`
it) and reads the inode from the disk,
if it has not already been read.
`iunlock`
releases the lock on the inode.
Separating acquisition of inode pointers from locking
helps avoid deadlock in some situations, for example during
directory lookup.
Multiple processes can hold a C pointer to an inode
returned by 
`iget`,
but only one process can lock the inode at a time.

The inode table only stores inodes to which kernel code
or data structures hold C pointers.
Its main job is synchronizing access by multiple processes.
The inode table also happens to cache frequently-used inodes, but
caching is secondary; if an inode is used frequently, the buffer cache will probably
keep it in memory.
Code that modifies an in-memory inode writes it to disk with
`iupdate`.

To allocate a new inode (for example, when creating a file),
xv6 calls
`ialloc`
سطر `kernel/fs.c:/^ialloc/`.
`ialloc`
is similar to
`balloc`:
it loops over the inode structures on the disk, one block at a time,
looking for one that is marked free.
When it finds one, it claims it by writing the new 
`type`
to the disk and then returns an entry from the inode table
with the tail call to 
`iget`
سطر `kernel/fs.c:/return.iget\(dev..inum\)/`.
The correct operation of
`ialloc`
depends on the fact that only one process at a time
can be holding a reference to 
`bp`:
`ialloc`
can be sure that some other process does not
simultaneously see that the inode is available
and try to claim it.

`iget`
سطر `kernel/fs.c:/^iget/`
looks through the inode table for an active entry (`ip->ref`
`>`
`0`)
with the desired device and inode number.
If it finds one, it returns a new reference to that inode
\linerefs{kernel/fs.c:/^....if..ip->ref.>.0/,/^....}/}.
As
`iget`
scans, it records the position of the first empty slot
\linerefs{kernel/fs.c:/^....if..empty.==.0/,/empty.=.ip/},
which it uses if it needs to allocate a table entry.

Code must lock the inode using
`ilock`
before reading or writing its metadata or content.
`ilock`
سطر `kernel/fs.c:/^ilock/`
uses a sleep-lock for this purpose.
Once
`ilock`
has exclusive access to the inode, it reads the inode
from disk (more likely, the buffer cache) if needed.
The function
`iunlock`
سطر `kernel/fs.c:/^iunlock/`
releases the sleep-lock,
which may cause any processes sleeping
to be woken up.

`iput`
سطر `kernel/fs.c:/^iput/`
releases a C pointer to an inode
by decrementing the reference count
سطر `kernel/fs.c:/^..ip->ref--/`.
If this is the last reference, the inode's
slot in the inode table is now free and can be re-used
for a different inode.

If 
`iput`
sees that there are no C pointer references to an inode
and that the inode has no links to it (occurs in no
directory), then the inode and its data blocks must
be freed.
`iput`
calls
`itrunc`
to truncate the file to zero bytes, freeing the data blocks;
sets the inode type to 0 (unallocated);
and writes the inode to disk
سطر `kernel/fs.c:/inode.has.no.links.and/`.

The locking protocol in 
`iput`
in the case in which it frees the inode deserves a closer look.
One danger is that a concurrent thread might be waiting in
`ilock`
to use this inode (e.g., to read a file or list a directory),
and won't be prepared to find that the inode is no longer
allocated. This can't happen because there is no way for
a system call to get a pointer to an in-memory inode if it has
no links to it and 
`ip->ref`
is one. That one reference is the reference owned by the
thread calling
`iput`.
The other main danger is that a concurrent call to
`ialloc`
might choose the same inode that
`iput`
is freeing.
This can happen only after the
`iupdate`
writes the disk so that the inode has type zero.
This race is benign; the allocating thread will politely wait
to acquire the inode's sleep-lock before reading or writing
the inode, at which point
`iput`
is done with it.

`iput()`
can write to the disk.  This means that any system call that uses the file
system may write to the disk, because the system call may be the last one having
a reference to the file. Even calls like
`read()`
that appear to be read-only, may end up calling
`iput().`
This, in turn, means that even read-only system calls
must be wrapped in transactions if they use the file system.

There is a challenging interaction between
`iput()`
and crashes.
`iput()`
doesn't truncate a file immediately when the link count for the file
drops to zero, because some process might still hold a reference to the inode in
memory: a process might still be reading and writing to the file, because it
successfully opened it. But, if a crash happens before the last process closes
the file descriptor for the file, then the file will be marked allocated on disk
but no directory entry will point to it.

File systems handle this case in one of two ways. The simple solution is that on
recovery, after reboot, the file system scans the whole file system for files
that are marked allocated, but have no directory entry pointing to them.  If any
such file exists, then it can free those files.

The second solution doesn't require scanning the file system.  In this solution,
the file system records on disk (e.g., in the super block) the inode inumber of
a file whose link count drops to zero but whose reference count isn't zero.  If
the file system removes the file when its reference count reaches 0, then it
updates the on-disk list by removing that inode from the list. On recovery, the
file system frees any file in the list.

== Code: Inode content

\begin{figure}[t]
\center
\includegraphics[scale=0.5]{fig/inode.pdf}
\caption{The representation of a file on disk.}
<fig:inode>
\end{figure}

The on-disk inode structure,
`struct dinode`,
contains a size and an array of block numbers (see 
Figure~@fig:inode).
The inode data is found in the blocks listed
in the
`dinode` 's
`addrs`
array.
The first
`NDIRECT`
blocks of data are listed in the first
`NDIRECT`
entries in the array; these blocks are called 
_direct blocks_.
The next 
`NINDIRECT`
blocks of data are listed not in the inode
but in a data block called the
_indirect block_.
The last entry in the
`addrs`
array gives the address of the indirect block.
Thus the first 12 kB (
`NDIRECT` 
`x`
`BSIZE`)
bytes of a file can be loaded from blocks listed in the inode,
while the next
`256` kB (
`NINDIRECT`
`x`
`BSIZE`)
bytes can only be loaded after consulting the indirect block.
This is a good on-disk representation but a 
complex one for clients.
The function
`bmap`
manages the representation so that higher-level routines, such as
`readi`
and
`writei`,
which we will see shortly, do not need to manage this complexity.
`bmap`
returns the disk block number of the
`bn`'th
data block for the inode
`ip`.
If
`ip`
does not have such a block yet,
`bmap`
allocates one.

The function
`bmap`
سطر `kernel/fs.c:/^bmap/`
begins by picking off the easy case: the first 
`NDIRECT`
blocks are listed in the inode itself
\linerefs{kernel/fs.c:/^..if..bn.<.NDIRECT/,/^..}/}.
The next 
`NINDIRECT`
blocks are listed in the indirect block at
`ip->addrs[NDIRECT]`.
`bmap`
reads the indirect block
سطر `kernel/fs.c:/bp.=.bread.ip->dev..addr/`
and then reads a block number from the right 
position within the block
سطر `kernel/fs.c:/a.=..uint.\*.bp->data/`.
If the block number exceeds
`NDIRECT+NINDIRECT`,
`bmap` 
panics; 
`writei`
contains the check that prevents this from happening
سطر `kernel/fs.c:/off...n...MAXFILE...BSIZE/`.

`bmap`
allocates blocks as needed.
An
`ip->addrs[]`
or indirect
entry of zero indicates that no block is allocated.
As
`bmap`
encounters zeros, it replaces them with the numbers of fresh blocks,
allocated on demand
\linerefs{kernel/fs.c:/^....if...addr.=.*==.0/,/./}
\linerefs{kernel/fs.c:/^....if...addr.*NDIRECT.*==.0/,/./}.

`itrunc`
frees a file's blocks, resetting the inode's size to zero.
`itrunc`
سطر `kernel/fs.c:/^itrunc/`
starts by freeing the direct blocks
\linerefs{kernel/fs.c:/^..for..i.=.0.*NDIRECT/,/^..}/},
then the ones listed in the indirect block
\linerefs{kernel/fs.c:/^....for..j.=.0.*NINDIRECT/,/^....}/},
and finally the indirect block itself
\linerefs{kernel/fs.c:/^....bfree.*NDIRECT/,/./}.

`bmap`
makes it easy for
`readi`
and
`writei` 
to get at an inode's data.
`readi`
سطر `kernel/fs.c:/^readi/`
starts by
making sure that the offset and count are not 
beyond the end of the file.
Reads that start beyond the end of the file return an error
\linerefs{kernel/fs.c:/^..if..off.>.ip->size/,/./}
while reads that start at or cross the end of the file 
return fewer bytes than requested
\linerefs{kernel/fs.c:/^..if..off.\+.n.>.ip->size/,/./}.
The main loop processes each block of the file,
copying data from the buffer into 
`dst`
\linerefs{kernel/fs.c:/^..for..tot.=.0/,/^..}/}.
%%  NOTE: It is very hard to write line references
%%  for writei because so many of the lines are identical
%%  to those in readi.  Luckily, identical lines probably
%%  don't need to be commented upon.
`writei`
سطر `kernel/fs.c:/^writei/`
is identical to
`readi`,
with three exceptions:
writes that start at or cross the end of the file
grow the file, up to the maximum file size
\linerefs{kernel/fs.c:/^..if..off.\+.n.>.MAXFILE/,/./};
the loop copies data into the buffers instead of out
سطر `kernel/fs.c:/either.copyin.*bp->data/`;
and if the write has extended the file,
`writei`
must update its size
\linerefs{kernel/fs.c:/^..if..off.>.ip->size\)/,/./}.

== Code: directory layer

A directory is implemented internally much like a file.
Its inode has type
`T_DIR`
and its data is a sequence of directory entries.
Each entry is a
`struct dirent`
سطر `kernel/fs.h:/^struct.dirent/`,
which contains a name and an inode number.
The name is at most
`DIRSIZ`
(14) characters;
if shorter, it is terminated by a NULL (0) byte.
Directory entries with inode number zero are free.

The reason why a directory entry contains the file's i-number rather
than the named file's entire i-node is to support ``links'' created by
the `link()` system call. An i-node can be referred to in multiple
directories, and thus under multiple path names; each of an i-node's
names (i.e. directory entries) is called a link. Because of the
possibility of an i-node being named in multiple directories, it is
not convenient for the i-node to be stored in any one of those
directories. The possibility of multiple links is also the reason for
the `nlink` field in the i-node. The fact that i-nodes are stored
separately from directories also allows sensible handling of a file
being unlinked (removed) while some process has a file descriptor
referring to the file: the file descriptor refers to the i-node, not
to any directory entry, so the file descriptor will still work even
though the file (really, just its name) has been removed.

The function
`dirlookup`
سطر `kernel/fs.c:/^dirlookup/`
searches a directory for an entry with the given name.
If it finds one, it returns a pointer to the corresponding inode, unlocked,
and sets 
`*poff`
to the byte offset of the entry within the directory,
in case the caller wishes to edit it.
If
`dirlookup`
finds an entry with the right name,
it updates
`*poff`
and returns an unlocked inode
obtained via
`iget`.
`dirlookup`
is the reason that 
`iget`
returns unlocked inodes.
The caller has locked
`dp`,
so if the lookup was for
`.`,
an alias for the current directory,
attempting to lock the inode before
returning would try to re-lock
`dp`
and deadlock.
(There are more complicated deadlock scenarios involving
multiple processes and
`..`,
an alias for the parent directory;
`.`
is not the only problem.)
The caller can unlock
`dp`
and then lock
`ip`,
ensuring that it only holds one lock at a time.

== Code: Path names

Path name lookup involves a succession of calls to
`dirlookup`,
one for each path component.
`namei`
سطر `kernel/fs.c:/^namei/`
evaluates 
`path`
and returns the corresponding 
`inode`.
The function
`nameiparent`
is a variant: it stops before the last element, returning the 
inode of the parent directory and copying the final element into
`name`.
Both call the generalized function
`namex`
to do the real work.

`namex`
سطر `kernel/fs.c:/^namex/`
starts by deciding where the path evaluation begins.
If the path begins with a slash, evaluation begins at the root;
otherwise, the current directory
\linerefs{kernel/fs.c:/..if..\*path.==....\)/,/idup/}.
Then it uses
`skipelem`
to consider each element of the path in turn
سطر `kernel/fs.c:/while.*skipelem/`.
Each iteration of the loop must look up 
`name`
in the current inode
`ip`.
The iteration begins by locking
`ip`
and checking that it is a directory.
If not, the lookup fails
\linerefs{kernel/fs.c:/^....ilock.ip/,/^....}/}.
(Locking
`ip`
is necessary not because 
`ip->type`
can change underfoot—it can't—but because
until 
`ilock`
runs,
`ip->type`
is not guaranteed to have been loaded from disk.)
If the call is 
`nameiparent`
and this is the last path element, the loop stops early,
as per the definition of
`nameiparent`;
the final path element has already been copied
into
`name`,
so
`namex`
need only
return the unlocked
`ip`
\linerefs{kernel/fs.c:/^....if..nameiparent/,/^....}/}.
Finally, the loop looks for the path element using
`dirlookup`
and prepares for the next iteration by setting
`ip = next`
\linerefs{kernel/fs.c:/^....if...next.*dirlookup/,/^....ip.=.next/}.
When the loop runs out of path elements, it returns
`ip`.

The procedure
`namex`
may take a long time to complete: it could involve several disk operations to
read inodes and directory blocks for the directories traversed in the pathname
(if they are not in the buffer cache).  Xv6 is carefully designed so that if an
invocation of
`namex`
by one kernel thread is blocked on a disk I/O, another kernel thread looking up
a different pathname can proceed concurrently.
`namex`
locks each directory in the path separately so that lookups in different
directories can proceed in parallel.

This concurrency introduces some challenges. For example, while one kernel
thread is looking up a pathname another kernel thread may be changing the
directory tree by unlinking a directory.  A potential risk is that a lookup
may be searching a directory that has been deleted by another kernel thread and
its blocks have been re-used for another directory or file.

Xv6 avoids such races.  For example, when executing
`dirlookup`
in
`namex`,
the lookup thread holds the lock on the directory and
`dirlookup`
returns an inode that was obtained using
`iget`.
`iget`
increases the reference count of the inode.  Only after receiving the
inode from
`dirlookup`
does
`namex`
release the lock on the directory.  Now another thread may unlink the inode from
the directory but xv6 will not delete the inode yet, because the reference count
of the inode is still larger than zero.

== File descriptor layer

A cool aspect of the Unix interface is that most resources in Unix are
represented as files, including devices such as the console, pipes, and of
course, real files.  The file descriptor layer is the layer that achieves this
uniformity.

Xv6 gives each process its own table of open files, or
file descriptors, as we saw in
Chapter~@CH:UNIX.
Each open file is represented by a
`struct file`
سطر `kernel/file.h:/^struct.file/`,
which is a wrapper around either an inode or a pipe,
plus an I/O offset.
Each call to 
`open`
creates a new open file (a new
`struct`
`file`):
if multiple processes open the same file independently,
the different instances will have different I/O offsets.
On the other hand, a single open file
(the same
`struct`
`file`)
can appear
multiple times in one process's file table
and also in the file tables of multiple processes.
This would happen if one process used
`open`
to open the file and then created aliases using
`dup`
or shared it with a child using
`fork`.
A reference count tracks the number of references to
a particular open file.
A file can be open for reading or writing or both.
The
`readable`
and
`writable`
fields track this.

All the open files in the system are kept in a global file table,
the 
`ftable`.
The file table
has functions to allocate a file
(`filealloc`),
create a duplicate reference
(`filedup`),
release a reference
(`fileclose`),
and read and write data
(`fileread`
and 
`filewrite`).

The first three follow the now-familiar form.
`filealloc`
سطر `kernel/file.c:/^filealloc/`
scans the file table for an unreferenced file
(`f->ref`
`==`
`0`)
and returns a new reference;
`filedup`
سطر `kernel/file.c:/^filedup/`
increments the reference count;
and
`fileclose`
سطر `kernel/file.c:/^fileclose/`
decrements it.
When a file's reference count reaches zero,
`fileclose`
releases the underlying pipe or inode,
according to the type.

== Code: System calls

With the functions that the lower layers provide, the implementation of most
system calls is trivial
(see
`kernel/sysfile.c`).
There are a few calls that
deserve a closer look.

The functions
`sys_link`
and
`sys_unlink`
edit directories, creating or removing references to inodes.
They are another good example of the power of using 
transactions. 
`sys_link`
سطر `kernel/sysfile.c:/^sys_link/`
begins by fetching its arguments, two strings
`old`
and
`new`
سطر `kernel/sysfile.c:/argstr.*old.*new/`.
Assuming 
`old`
exists and is not  a directory
\linerefs{kernel/sysfile.c:/namei.old/,/ip->n/},
`sys_link`
increments its 
`ip->nlink`
count.
Then
`sys_link`
calls
`nameiparent`
to find the parent directory and final path element of
`new` 
سطر `kernel/sysfile.c:/nameiparent.new/`
and creates a new directory entry pointing at
`old` 's
inode
سطر `kernel/sysfile.c:/\|\| dirlink/`.
The new parent directory must exist and
be on the same device as the existing inode:
inode numbers only have a unique meaning on a single disk.
If an error like this occurs, 
`sys_link`
must go back and decrement
`ip->nlink`.

Transactions simplify the implementation because it requires updating multiple
disk blocks, but we don't have to worry about the order in which we do
them. They either will all succeed or none.
For example, without transactions, updating
`ip->nlink`
before creating a link, would put the file system temporarily in an unsafe
state, and a crash in between could result in havoc.
With transactions we don't have to worry about this.

`sys_link`
creates a new name for an existing inode.
The function
`create`
سطر `kernel/sysfile.c:/^create/`
creates a new name for a new inode.
It is a generalization of the three file creation
system calls:
`open`
with the
`O_CREATE`
flag makes a new ordinary file,
`mkdir`
makes a new directory,
and
`mkdev`
makes a new device file.
Like
`sys_link`,
`create`
starts by calling
`nameiparent`
to get the inode of the parent directory.
It then calls
`dirlookup`
to check whether the name already exists
سطر `kernel/sysfile.c:/dirlookup.*[^=]=.0/`.
If the name does exist, 
`create`'s
behavior depends on which system call it is being used for:
`open`
has different semantics from 
`mkdir`
and
`mkdev`.
If
`create`
is being used on behalf of
`open`
(`type`
`==`
`T_FILE`)
and the name that exists is itself
a regular file,
then 
`open`
treats that as a success,
so
`create`
does too
سطر `kernel/sysfile.c:/^......return.ip/`.
Otherwise, it is an error
\linerefs{kernel/sysfile.c:/^......return.ip/+1,/return.0/}.
If the name does not already exist,
`create`
now allocates a new inode with
`ialloc`
سطر `kernel/sysfile.c:/ialloc/`.
If the new inode is a directory, 
`create`
initializes it with
`.`
and
`..`
entries.
Finally, now that the data is initialized properly,
`create`
can link it into the parent directory
سطر `kernel/sysfile.c:/if..dirlink\(dp/`.
`create`,
like
`sys_link`,
holds two inode locks simultaneously:
`ip`
and
`dp`.
There is no possibility of deadlock because
the inode
`ip`
is freshly allocated: no other process in the system
will hold 
`ip` 's
lock and then try to lock
`dp`.

Using
`create`,
it is easy to implement
`sys_open`,
`sys_mkdir`,
and
`sys_mknod`.
`sys_open`
سطر `kernel/sysfile.c:/^sys_open/`
is the most complex, because creating a new file is only
a small part of what it can do.
If
`open`
is passed the
`O_CREATE`
flag, it calls
`create`
سطر `kernel/sysfile.c:/create.*T_FILE/`.
Otherwise, it calls
`namei`
سطر `kernel/sysfile.c:/if...ip.=.namei.path/`.
`create`
returns a locked inode, but 
`namei`
does not, so
`sys_open`
must lock the inode itself.
This provides a convenient place to check that directories
are only opened for reading, not writing.
Assuming the inode was obtained one way or the other,
`sys_open`
allocates a file and a file descriptor
سطر `kernel/sysfile.c:/filealloc.*fdalloc/`
and then fills in the file
\linerefs{kernel/sysfile.c:/type.=.FD_INODE/,/writable/}.
Note that no other process can access the partially initialized file since it is only
in the current process's table.

== Real world

The buffer cache in a real-world operating system is significantly
more complex than xv6's, but it serves the same two purposes:
caching and synchronizing access to the disk.
Xv6's buffer cache uses a least recently used (LRU)
eviction policy; there are many more complex
policies that can be implemented, each good for some
workloads and not as good for others.
A more efficient LRU cache would eliminate the linked list,
instead using a hash table for lookups and a heap for LRU evictions.
Modern buffer caches are typically integrated with the
virtual memory system to support memory-mapped files.

Xv6 uses an on-disk layout of inodes and directories
similar to that of early UNIX;
this scheme has been remarkably persistent over the years.
BSD's UFS/FFS and Linux's ext2/ext3 use essentially the same data structures.
The most inefficient part of the file system layout is the directory,
which requires a linear scan over all the disk blocks during each lookup.
This is reasonable when directories are only a few disk blocks,
but is expensive for directories holding many files.
Microsoft Windows's NTFS, macOS's HFS, and Solaris's ZFS, just to name a few, implement
a directory as an on-disk balanced tree of blocks.
This is complicated but guarantees logarithmic-time directory lookups.

Xv6 requires that the file system
fit on one disk device and not change in size.
As large databases and multimedia files drive storage
requirements ever higher, operating systems are developing ways
to eliminate the ``one disk per file system'' bottleneck.
The basic approach is to combine many disks into a single
logical disk.  Hardware solutions such as RAID are still the 
most popular, but the current trend is moving toward implementing
as much of this logic in software as possible.
These software implementations typically 
allow rich functionality like growing or shrinking the logical
device by adding or removing disks on the fly.
Of course, a storage layer that can grow or shrink on the fly
requires a file system that can do the same: the fixed-size array
of inode blocks used by xv6 would not work well
in such environments.
Separating disk management from the file system may be
the cleanest design, but the complex interface between the two
has led some systems, like Sun's ZFS, to combine them.

Xv6's file system lacks many other features of modern file systems; for example,
it lacks support for snapshots and incremental backup.

== Exercises

\begin{enumerate}

\item Why panic in
`balloc` ?
Can xv6 recover?

\item Why panic in
`ialloc` ?
Can xv6 recover?

\item Why doesn't
`filealloc`
panic when it runs out of files?
Why is this more common and therefore worth handling?

\item Suppose the file corresponding to 
`ip`
gets unlinked by another process
between 
`sys_link` 's
calls to 
`iunlock(ip)`
and
`dirlink`.
Will the link be created correctly?
Why or why not?

\item
`create`
makes four function calls (one to
`ialloc`
and three to
`dirlink`)
that it requires to succeed.
If any doesn't,
`create`
calls
`panic`.
Why is this acceptable?
Why can't any of those four calls fail?

\item
`sys_chdir`
calls
`iunlock(ip)`
before
`iput(cp->cwd)`,
which might try to lock
`cp->cwd`,
yet postponing
`iunlock(ip)`
until after the
`iput`
would not cause deadlocks.
Why not?

\item Implement the
`lseek`
system call.  Supporting
`lseek`
will also require that you modify
`filewrite`
to fill holes in the file with zero if
`lseek`
sets
`off`
beyond
`f->ip->size.`

\item Add
`O_TRUNC`
and
`O_APPEND`
to
`open`,
so that
`>`
and
`>>`
operators work in the shell.

\item Modify the file system to support symbolic links.

\item Modify the file system to support named pipes.

\item Modify the file and VM system to support memory-mapped files.

\end{enumerate}
