cmake_minimum_required(VERSION 3.21)

project(ish_go_native VERSION 1.0 LANGUAGES C ASM)

# Force a clean, non-cached native build so source fixes in upstream/ish-ios
# are always recompiled (no ccache / compiler-launcher reuse of stale objects).
unset(CMAKE_C_COMPILER_LAUNCHER CACHE)
set(CMAKE_C_COMPILER_LAUNCHER "")
unset(CMAKE_CXX_COMPILER_LAUNCHER CACHE)
set(CMAKE_CXX_COMPILER_LAUNCHER "")

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY "${CMAKE_CURRENT_BINARY_DIR}/lib")

find_package(Python3 COMPONENTS Interpreter REQUIRED)
find_package(Threads REQUIRED)
if(NOT ANDROID)
    find_package(ZLIB REQUIRED)
    set(ISH_ZLIB_LIB ZLIB::ZLIB)
    set(ISH_ANDROID_LOG_LIB)
else()
    set(ISH_ZLIB_LIB z)
    set(ISH_ANDROID_LOG_LIB log)
endif()

set(ISH_ROOT "${CMAKE_CURRENT_LIST_DIR}/../..")
set(ISH_SOURCE_DIR "${ISH_ROOT}/upstream/ish-ios")
set(ISH_CORE_BUILD_DIR "${CMAKE_CURRENT_BINARY_DIR}/ish-core")
file(MAKE_DIRECTORY "${ISH_CORE_BUILD_DIR}/vdso")
file(MAKE_DIRECTORY "${CMAKE_CURRENT_BINARY_DIR}/vdso")

if(ANDROID)
    set(ISH_PLATFORM android)
else()
    set(ISH_PLATFORM linux)
endif()

if(CMAKE_ANDROID_ARCH_ABI STREQUAL "arm64-v8a" OR CMAKE_SYSTEM_PROCESSOR MATCHES "^(aarch64|arm64|ARM64)$")
    set(ISH_GADGET_ARCH aarch64)
elseif(CMAKE_ANDROID_ARCH_ABI STREQUAL "x86_64" OR CMAKE_SYSTEM_PROCESSOR MATCHES "^(x86_64|amd64|AMD64)$")
    set(ISH_GADGET_ARCH x86_64)
else()
    message(FATAL_ERROR "Unsupported Asbestos host architecture: ${CMAKE_SYSTEM_PROCESSOR}; ABI=${CMAKE_ANDROID_ARCH_ABI}")
endif()

if(ANDROID)
    set(ISH_VDSO_CLANG "${CMAKE_C_COMPILER}")
else()
    find_program(ISH_VDSO_CLANG NAMES clang clang-18 clang-17 REQUIRED)
endif()

set(ISH_VDSO_FILE "${CMAKE_CURRENT_BINARY_DIR}/vdso/libvdso.so.elf")
add_custom_command(
    OUTPUT "${ISH_VDSO_FILE}"
    COMMAND "${ISH_VDSO_CLANG}" -target i386-linux -fuse-ld=lld
            -o "${ISH_VDSO_FILE}"
            "${ISH_SOURCE_DIR}/vdso/vdso.S"
            "${ISH_SOURCE_DIR}/vdso/vdso.c"
            -nostdlib
            -Wl,-T,${ISH_SOURCE_DIR}/vdso/vdso.lds
            -Wl,--hash-style,sysv
            -shared -fPIC
    DEPENDS
        "${ISH_SOURCE_DIR}/vdso/vdso.S"
        "${ISH_SOURCE_DIR}/vdso/vdso.c"
        "${ISH_SOURCE_DIR}/vdso/vdso.lds"
    VERBATIM
)
add_custom_target(ish_vdso DEPENDS "${ISH_VDSO_FILE}")

set(ISH_OFFSETS_FILE "${ISH_CORE_BUILD_DIR}/cpu-offsets.h")
add_custom_command(
    OUTPUT "${ISH_OFFSETS_FILE}"
    COMMAND "${Python3_EXECUTABLE}" "${ISH_SOURCE_DIR}/tools/generate-offsets.py"
            --compiler "${CMAKE_C_COMPILER}"
            --source "${ISH_SOURCE_DIR}/asbestos/offsets.c"
            --include-root "${ISH_SOURCE_DIR}"
            --staticdefine "${ISH_SOURCE_DIR}/tools/staticdefine.h"
            --output "${ISH_OFFSETS_FILE}"
    DEPENDS
        "${ISH_SOURCE_DIR}/tools/generate-offsets.py"
        "${ISH_SOURCE_DIR}/tools/staticdefine.h"
        "${ISH_SOURCE_DIR}/asbestos/offsets.c"
    VERBATIM
)
add_custom_target(ish_offsets DEPENDS "${ISH_OFFSETS_FILE}")

set(ISH_EMU_SOURCES
    "${ISH_SOURCE_DIR}/emu/tlb.c"
    "${ISH_SOURCE_DIR}/emu/fpu.c"
    "${ISH_SOURCE_DIR}/emu/vec.c"
    "${ISH_SOURCE_DIR}/emu/mmx.c"
    "${ISH_SOURCE_DIR}/emu/float80.c"
    "${ISH_SOURCE_DIR}/asbestos/asbestos.c"
    "${ISH_SOURCE_DIR}/asbestos/gen.c"
    "${ISH_SOURCE_DIR}/asbestos/helpers.c"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/entry.S"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/memory.S"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/control.S"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/math.S"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/bits.S"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/string.S"
    "${ISH_SOURCE_DIR}/asbestos/gadgets-${ISH_GADGET_ARCH}/misc.S"
)
add_library(ish_emu STATIC ${ISH_EMU_SOURCES})
add_dependencies(ish_emu ish_offsets)
target_include_directories(ish_emu PRIVATE "${ISH_SOURCE_DIR}" "${ISH_CORE_BUILD_DIR}")
target_compile_definitions(ish_emu PRIVATE ENGINE_ASBESTOS=1)
target_compile_options(ish_emu PRIVATE
    -Wno-switch -Wno-unused-parameter -Wno-sign-compare
    -Wno-pointer-sign -Wno-implicit-fallthrough
)

add_library(ish_sqlite STATIC "${ISH_SOURCE_DIR}/deps/sqlite/sqlite3.c")
target_include_directories(ish_sqlite PUBLIC "${ISH_SOURCE_DIR}/deps/sqlite")
target_compile_definitions(ish_sqlite PRIVATE SQLITE_THREADSAFE=1)

add_library(ish_fakefs STATIC
    "${ISH_SOURCE_DIR}/fs/fake-db.c"
    "${ISH_SOURCE_DIR}/fs/fake-migrate.c"
    "${ISH_SOURCE_DIR}/fs/fake-rebuild.c"
)
target_include_directories(ish_fakefs PRIVATE "${ISH_SOURCE_DIR}" "${ISH_SOURCE_DIR}/deps/sqlite")
target_compile_definitions(ish_fakefs PRIVATE ENGINE_ASBESTOS=1)
target_link_libraries(ish_fakefs PUBLIC ish_sqlite)

set(ISH_KERNEL_SOURCES
    "${ISH_SOURCE_DIR}/kernel/init.c"
    "${ISH_SOURCE_DIR}/kernel/errno.c"
    "${ISH_SOURCE_DIR}/kernel/calls.c"
    "${ISH_SOURCE_DIR}/kernel/memory.c"
    "${ISH_SOURCE_DIR}/kernel/user.c"
    "${ISH_SOURCE_DIR}/kernel/vdso.c"
    "${ISH_SOURCE_DIR}/kernel/task.c"
    "${ISH_SOURCE_DIR}/kernel/group.c"
    "${ISH_SOURCE_DIR}/kernel/log.c"
    "${ISH_SOURCE_DIR}/kernel/fork.c"
    "${ISH_SOURCE_DIR}/kernel/exec.c"
    "${ISH_SOURCE_DIR}/kernel/exit.c"
    "${ISH_SOURCE_DIR}/kernel/time.c"
    "${ISH_SOURCE_DIR}/kernel/mmap.c"
    "${ISH_SOURCE_DIR}/kernel/uname.c"
    "${ISH_SOURCE_DIR}/kernel/tls.c"
    "${ISH_SOURCE_DIR}/kernel/futex.c"
    "${ISH_SOURCE_DIR}/kernel/getset.c"
    "${ISH_SOURCE_DIR}/kernel/signal.c"
    "${ISH_SOURCE_DIR}/kernel/resource.c"
    "${ISH_SOURCE_DIR}/kernel/random.c"
    "${ISH_SOURCE_DIR}/kernel/misc.c"
    "${ISH_SOURCE_DIR}/kernel/eventfd.c"
    "${ISH_SOURCE_DIR}/kernel/ipc.c"
    "${ISH_SOURCE_DIR}/kernel/ptrace.c"
    "${ISH_SOURCE_DIR}/kernel/fs.c"
    "${ISH_SOURCE_DIR}/kernel/fs_info.c"
    "${ISH_SOURCE_DIR}/fs/mount.c"
    "${ISH_SOURCE_DIR}/fs/fd.c"
    "${ISH_SOURCE_DIR}/fs/inode.c"
    "${ISH_SOURCE_DIR}/fs/stat.c"
    "${ISH_SOURCE_DIR}/fs/dir.c"
    "${ISH_SOURCE_DIR}/fs/generic.c"
    "${ISH_SOURCE_DIR}/fs/path.c"
    "${ISH_SOURCE_DIR}/fs/real.c"
    "${ISH_SOURCE_DIR}/fs/fake.c"
    "${ISH_SOURCE_DIR}/fs/proc.c"
    "${ISH_SOURCE_DIR}/fs/proc/entry.c"
    "${ISH_SOURCE_DIR}/fs/proc/ish.c"
    "${ISH_SOURCE_DIR}/fs/proc/root.c"
    "${ISH_SOURCE_DIR}/fs/proc/pid.c"
    "${ISH_SOURCE_DIR}/fs/dyndev.c"
    "${ISH_SOURCE_DIR}/fs/adhoc.c"
    "${ISH_SOURCE_DIR}/fs/sock.c"
    "${ISH_SOURCE_DIR}/fs/pipe.c"
    "${ISH_SOURCE_DIR}/fs/sockrestart.c"
    "${ISH_SOURCE_DIR}/fs/lock.c"
    "${ISH_SOURCE_DIR}/fs/dev.c"
    "${ISH_SOURCE_DIR}/fs/mem.c"
    "${ISH_SOURCE_DIR}/fs/tty.c"
    "${ISH_SOURCE_DIR}/fs/tty-real.c"
    "${ISH_SOURCE_DIR}/fs/pty.c"
    "${ISH_SOURCE_DIR}/fs/tmp.c"
    "${ISH_SOURCE_DIR}/fs/poll.c"
    "${ISH_SOURCE_DIR}/kernel/poll.c"
    "${ISH_SOURCE_DIR}/kernel/epoll.c"
    "${ISH_SOURCE_DIR}/util/timer.c"
    "${ISH_SOURCE_DIR}/util/sync.c"
    "${ISH_SOURCE_DIR}/util/fifo.c"
    "${ISH_SOURCE_DIR}/util/fchdir.c"
    "${ISH_SOURCE_DIR}/platform/${ISH_PLATFORM}.c"
)
add_library(ish_kernel STATIC ${ISH_KERNEL_SOURCES})
add_dependencies(ish_kernel ish_vdso)
target_include_directories(ish_kernel PRIVATE "${ISH_SOURCE_DIR}" "${ISH_SOURCE_DIR}/deps/sqlite" "${ISH_CORE_BUILD_DIR}")
target_compile_definitions(ish_kernel PRIVATE ENGINE_ASBESTOS=1 LOG_HANDLER_DPRINTF=1)
target_compile_options(ish_kernel PRIVATE
    -Wno-switch -Wno-unused-parameter -Wno-sign-compare
    -Wno-pointer-sign -Wno-implicit-fallthrough
)
target_link_libraries(ish_kernel PUBLIC ish_emu ish_fakefs ish_sqlite Threads::Threads)

add_library(ish_core_session STATIC "${ISH_SOURCE_DIR}/app/core/CoreSession.c")
target_include_directories(ish_core_session PUBLIC
    "${ISH_SOURCE_DIR}/app/core"
    "${ISH_SOURCE_DIR}"
    "${ISH_SOURCE_DIR}/deps/sqlite"
    "${ISH_CORE_BUILD_DIR}"
)
target_compile_definitions(ish_core_session PRIVATE ENGINE_ASBESTOS=1 LOG_HANDLER_DPRINTF=1 ISH_CORE_SESSION=1)
if(NOT ANDROID)
    target_compile_definitions(ish_core_session PRIVATE __ANDROID__=1 ISH_CORE_HOST=1)
endif()
target_compile_options(ish_core_session PRIVATE
    -Wno-switch -Wno-unused-parameter -Wno-sign-compare
    -Wno-pointer-sign -Wno-implicit-fallthrough
)
target_link_libraries(ish_core_session PUBLIC ish_kernel ish_emu ish_fakefs ish_sqlite Threads::Threads ${ISH_ZLIB_LIB} ${ISH_ANDROID_LOG_LIB} m)

add_library(ish_native INTERFACE)
target_link_libraries(ish_native INTERFACE ish_core_session)
target_include_directories(ish_native INTERFACE "${ISH_SOURCE_DIR}/app/core")

add_executable(ish-core-smoke "${ISH_SOURCE_DIR}/main.c")
target_include_directories(ish-core-smoke PRIVATE "${ISH_SOURCE_DIR}" "${ISH_CORE_BUILD_DIR}")
target_link_libraries(ish-core-smoke PRIVATE ish_core_session ish_kernel ish_emu ish_fakefs ish_sqlite Threads::Threads ${ISH_ZLIB_LIB} ${ISH_ANDROID_LOG_LIB} m)
