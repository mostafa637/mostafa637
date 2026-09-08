#import "../template/listings.typ": *

= Interrupts and device drivers <CH:INTERRUPT>

A
_driver_
is the code in an operating system that manages a particular device:
it configures the device hardware,
tells the device to perform operations,
handles the resulting interrupts,
and interacts with processes using the device.
Driver code can be tricky
because a driver executes concurrently with the device,
and often concurrently with processes using the device.  In
addition, the driver must understand the device's hardware interface,
which can be complex and poorly documented.

Devices that need attention from the operating system can usually be
configured to generate interrupts, which are one type of trap.
The kernel trap handling code recognizes when a device
has raised an interrupt and calls the driver's interrupt handler;
in xv6, this dispatch happens in `devintr` سطر `kernel/trap.c:/^devintr/`.

Many device drivers execute code in two contexts: a _bottom
half_ that runs in a process's kernel thread, and a _top
half_ that executes at interrupt time. The bottom half
is called via system calls such as `read` and `write` that want
the device to perform I/O. This code may ask the hardware to start an
operation (e.g., ask the disk to read a block); then the code waits for
the operation to complete. Eventually the device completes the
operation and raises an interrupt. The driver's interrupt handler,
acting as the top half,
figures out what operation has completed, wakes up a waiting
process if appropriate, and tells the hardware to start work
on the next operation, if any.

== Code: Console input

The console driver `kernel/console.c`
is a simple illustration of driver structure. The
console driver accepts characters typed by a human, via the _UART_
serial-port hardware attached to the RISC-V. The console driver accumulates a
line of input at a time, processing special input characters such as
backspace and control-u. User processes, such as the shell, use
the `read` system call to fetch lines of input from the console.
When you type input to xv6 in QEMU, your keystrokes are delivered to
xv6 by way of QEMU's simulated UART hardware.

The UART hardware that the driver talks to is a 16550
chip~\cite{ns16550a} emulated by QEMU. On a real computer, a 16550
would manage an RS232 serial link connecting to a terminal or other
computer. When running QEMU, it's connected to your keyboard and
display.

The UART hardware appears to software as a set of _memory-mapped_
control registers. That is, there are some physical addresses that 
are connected to the UART device, so that loads and stores
interact with the device hardware rather than RAM.
The memory-mapped addresses for the UART start at 0x10000000, or `UART0`
سطر `kernel/memlayout.h:/UART0.+0x/`.
There are a handful of UART control registers, each the width
of a byte. Their offsets from `UART0` are defined in
سطر `kernel/uart.c:/define.RHR/`. For example, the
`LSR` register contains bits that indicate whether input
characters are waiting to be read by the driver. These
characters (if any) are available for reading from the
`RHR` register. Each time one is read, the UART hardware
deletes it from an internal FIFO of waiting characters, and
clears the ``ready'' bit in `LSR` when the FIFO is empty.
To transmit, the driver writes a byte to the `THR` register,
which causes the UART to append the byte to a FIFO of bytes that the
UART will send on the RS232 serial link.
The UART transmit and receive hardware are largely independent
of each other.

Xv6's `main` calls `consoleinit`
سطر `kernel/console.c:/^consoleinit/` to initialize the UART
hardware. This code configures the UART to generate 
a receive 
interrupt when the UART receives each byte of input, and
a _transmit complete_ interrupt each time the
UART finishes sending a byte of output سطر `kernel/uart.c:/^uartinit/`.

The xv6 shell reads from the console by way of a file descriptor
opened by `init.c` سطر `user/init.c:/open..console/`. Calls to
the `read` system call make their way through the kernel to `consoleread` سطر `kernel/console.c:/^consoleread/`. `consoleread` waits for input to arrive (via interrupts) and be
buffered in `cons.buf`, copies the input to user space, and (after
a whole line has arrived) returns to the user process. If the user
hasn't typed a full line yet, any reading processes will wait in the
`sleep` call
سطر `kernel/console.c:/sleep..cons/`
(Chapter~@CH:SLEEP explains the details of `sleep`).

When the user types a character, the UART hardware asks the RISC-V
to raise an interrupt, which activates
xv6's trap handler.
The trap handler calls `devintr`
سطر `kernel/trap.c:/^devintr/`,
which looks at the RISC-V `scause` register to discover that
the interrupt is from an external device.
Then it asks a hardware unit called the PLIC
~\cite{riscv:priv}
to tell it which device interrupted
سطر `kernel/trap.c:/plic.claim/`.
If it was the UART, `devintr` calls `uartintr`.

`uartintr`
سطر `kernel/uart.c:/^uartintr/`
reads any waiting input characters from the UART hardware
and hands them to `consoleintr`
سطر `kernel/console.c:/^consoleintr/`; it doesn't
wait for characters, since future input will raise a new interrupt.
The job of `consoleintr` is to accumulate input characters in
`cons.buf` 
until a whole line arrives.
`consoleintr` treats backspace and a few other characters
specially.
When a newline arrives, `consoleintr` wakes up a
waiting `consoleread` (if there is one).

Once woken, `consoleread` will observe a full line in `cons.buf`, copy it to user space, and return (via the system call
machinery) to user space.

A pattern to note is the decoupling of device activity from process
activity via buffering and interrupts. The console driver can process
input even when no process is waiting to read it; a subsequent read
will see the input. This decoupling can increase performance by
allowing processes to execute concurrently with device I/O, and is
particularly important when the device is slow (as with the UART) or
needs immediate attention (as with echoing typed characters). This
idea is sometimes called _I/O concurrency_.

== Code: Console output

A `write` system call on a file descriptor connected to the console
reaches
`consolewrite`
سطر `kernel/console.c:/^consolewrite/`,
which copies batches of bytes from user space
and hands each batch to `uartwrite`
سطر `kernel/uart.c:/^uartwrite/`.
Before writing each byte to the UART's THR register,
`uartwrite` must wait for the UART to be ready
to accept more output. Because the UART is relatively
slow, `uartwrite` waits using `sleep`
(which yields the CPU)
rather than a busy-loop.
When the UART is done sending the most recent byte,
it interrupts. The interrupt routine,
`uartintr`
سطر `kernel/uart.c:/^uartintr/`,
cooperates with `uartwrite`:
the latter sets the `tx\_busy` flag when
it sends each character, and waits for
the flag to be cleared;
the interrupt routine clears `tx\_busy`
and wakes up the writing thread
when the UART signals it is ready for more output.

== Concurrency in drivers

You may have noticed calls to `acquire` in `consoleread`
and in `consoleintr`. These calls acquire a lock, which protects
the console driver's data structures from concurrent access.
There are three concurrency dangers here: two processes on
different CPUs might call `consoleread` at the same time;
the hardware might ask a CPU to deliver a console (really 
UART) interrupt while that CPU is already executing inside
`consoleread`;
and the hardware might deliver a console interrupt on
a different CPU while `consoleread` is executing.
Chapter~@CH:LOCK explains how to use locks
to ensure that these dangers don't lead to incorrect results.

Another way in which concurrency requires care in drivers is that one
process may be waiting for input from a device, but the interrupt
signaling arrival of the input may arrive when a different process (or
no process at all) is running. Thus interrupt handlers are not allowed
to think about the process or code that they have interrupted. For
example, an interrupt handler cannot safely call `copyout` with
the current process's page table. Interrupt handlers typically do
relatively little work (e.g., just copy the input data to a buffer),
and wake up bottom-half code to do the rest.

== Timer interrupts

Xv6 uses timer interrupts to maintain its idea of the
current time and to switch among compute-bound processes. Timer
interrupts come from clock hardware attached to each RISC-V CPU. Xv6
programs each CPU's clock hardware to interrupt the CPU periodically.

Code in `start.c` 
سطر `kernel/start.c:/^timerinit/` sets some control bits
that allow supervisor-mode access to the timer control
registers, and then asks for the first timer interrupt.
The `time` control register contains a count that the
hardware increments at a steady rate; this serves as a
notion of the current time. The `stimecmp` register
contains a time at which the the CPU will raise a timer
interrupt; setting `stimecmp` to the current value
of `time` plus {\it x} will schedule an interrupt
{\it x} time units in the future. For `qemu`'s RISC-V
emulation, 1000000 time units is roughly a tenth of second.

Timer interrupts arrive via `usertrap` or `kerneltrap`
and `devintr`, like other device interrupts.
Timer interrupts arrive with `scause`'s low bits set to
five; `devintr` in `trap.c` detects this situation
and calls `clockintr`
سطر `kernel/trap.c:/clockintr/`.
The latter function increments `ticks`,
allowing the kernel to track the
passage of time. The increment occurs on only one CPU, to avoid time
passing faster if there are multiple CPUs.
`clockintr` wakes up any processes waiting in the `pause`
system call,
and schedules the next timer interrupt by writing
`stimecmp`.

`devintr` returns 2 for a timer interrupt
in order to indicate to `kerneltrap`
or `usertrap` that they should call `yield` so that
CPUs can be multiplexed among runnable processes.

The fact that kernel code can be interrupted by a timer interrupt that
forces a context switch via `yield` is part of the reason why
early code in `usertrap` is careful to save state such as `sepc` before enabling interrupts. These context switches also mean
that kernel code must be written in the knowledge that it may move
from one CPU to another without warning.

== Real world

Xv6, like many operating systems, allows interrupts and even context
switches (via `yield`) while executing in the kernel. The reason
for this is to retain quick response times during complex system calls
that run for a long time. However, as noted above, allowing interrupts
in the kernel is the source of some complexity; as a result, a few
operating systems allow interrupts only while executing user code.

Supporting all the devices on a typical computer in its full glory is
much work, because there are many devices, the devices have many
features, and the protocol between device and driver can be complex
and poorly documented. In many operating systems, the drivers account
for more code than the core kernel.

The UART driver retrieves data a byte at a time by reading the UART
control registers; this pattern is called _programmed I/O_, since
software is driving the data movement. Programmed I/O is simple, but
too slow to be used at high data rates. Devices that need to move lots
of data at high speed typically use _direct memory access (DMA)_.
DMA device hardware directly writes incoming data to RAM, and reads
outgoing data from RAM. Modern disk and network devices use DMA. A
driver for a DMA device would prepare data in RAM, and then use a
single write to a control register to tell the device to process the
prepared data.

Interrupts make sense when a device needs attention at unpredictable
times, and not too often. But interrupts have high CPU overhead. Thus
high speed devices, such as network and disk controllers, use tricks
that reduce the need for interrupts. One trick is to raise a single
interrupt for a whole batch of incoming or outgoing requests. Another
trick is for the driver to disable interrupts entirely, and to check
the device periodically to see if it needs attention. This technique
is called _polling_. Polling makes sense if the device performs
operations at a high rate, but it wastes CPU time if the device is mostly
idle. Some drivers dynamically switch between polling and interrupts
depending on the current device load.

The UART driver copies incoming data first to a buffer in the kernel,
and then to user space. This makes sense at low data rates, but such a
double copy can significantly reduce performance for devices that
generate or consume data very quickly. Some operating systems are able
to directly move data between user-space buffers and device hardware,
often with DMA.

As mentioned in Chapter~@CH:UNIX, the console appears to
applications as a regular file, and applications read input and write
output using the `read` and `write` system calls.
Applications may want to control aspects of a device that cannot be
expressed through the standard file system calls (e.g.,
enabling/disabling line buffering in the console driver).  Unix
operating systems provide an `ioctl` system call for such
cases.

Some uses of computers require ``real-time'' responses to external
events: responses guaranteed to occur within a bounded time. For
example, in safety-critical systems missing a deadline can lead to
disasters. Xv6 is not suitable for real-time settings. Among other
things, xv6's scheduler does not take into account real-time deadlines
when it decides what process to run next, and xv6 has long kernel code
paths with interrupts disabled, so that it may not respond to
interrupts quickly. A real-time operating system must not only fix
these problems, but also be structured in a way that allows analysis
of worst-case response times.

== Exercises

\begin{enumerate}

\item Modify `uart.c` to not use interrupts at all. You may need
to modify `console.c` as well.

\item Add a driver for an Ethernet card.

\end{enumerate}
