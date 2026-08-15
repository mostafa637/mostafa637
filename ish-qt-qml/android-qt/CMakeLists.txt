cmake_minimum_required(VERSION 3.21)

project(ish_qt_android VERSION 1.0 LANGUAGES C CXX ASM)

set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Qt6 6.11.1 REQUIRED COMPONENTS Core Gui Quick WebView WebChannel WebSockets)
find_package(Python3 COMPONENTS Interpreter REQUIRED)
find_package(Threads REQUIRED)
find_package(ZLIB REQUIRED)
qt_standard_project_setup(REQUIRES 6.8)

set(ISH_ROOT "${CMAKE_CURRENT_LIST_DIR}/..")
set(ISH_SOURCE_DIR "${ISH_ROOT}/upstream/ish-ios")
set(ISH_CORE_BUILD_DIR "${CMAKE_CURRENT_BINARY_DIR}/ish-core")
file(MAKE_DIRECTORY "${ISH_CORE_BUILD_DIR}/vdso")

if(ANDROID)
    set(ISH_PLATFORM android)
else()
    set(ISH_PLATFORM linux)
endif()

if(CMAKE_SYSTEM_PROCESSOR MATCHES "^(x86_64|amd64|AMD64)$")
    set(ISH_GADGET_ARCH x86_64)
elseif(CMAKE_SYSTEM_PROCESSOR MATCHES "^(aarch64|arm64|ARM64)$")
    set(ISH_GADGET_ARCH aarch64)
else()
    message(FATAL_ERROR "Unsupported Asbestos host architecture: ${CMAKE_SYSTEM_PROCESSOR}")
endif()

find_program(ISH_VDSO_CLANG NAMES clang clang-18 clang-17 REQUIRED)

# kernel/vdso.c uses a relative .incbin path (`vdso/libvdso.so.elf`).
# Keep the generated file below the target binary directory so both GCC and
# Clang resolve that path consistently on Linux and Android.
set(ISH_VDSO_FILE "${CMAKE_CURRENT_BINARY_DIR}/vdso/libvdso.so.elf")
file(MAKE_DIRECTORY "${CMAKE_CURRENT_BINARY_DIR}/vdso")
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
target_include_directories(ish_emu PRIVATE
    "${ISH_SOURCE_DIR}"
    "${ISH_CORE_BUILD_DIR}"
)
target_compile_definitions(ish_emu PRIVATE ENGINE_ASBESTOS=1)
target_compile_options(ish_emu PRIVATE
    -Wno-switch
    -Wno-unused-parameter
    -Wno-sign-compare
    -Wno-pointer-sign
    -Wno-implicit-fallthrough
)

add_library(ish_sqlite STATIC "${ISH_SOURCE_DIR}/deps/sqlite/sqlite3.c")
target_include_directories(ish_sqlite PUBLIC
    "${ISH_SOURCE_DIR}/deps/sqlite"
)
target_compile_definitions(ish_sqlite PRIVATE SQLITE_THREADSAFE=1)

add_library(ish_fakefs STATIC
    "${ISH_SOURCE_DIR}/fs/fake-db.c"
    "${ISH_SOURCE_DIR}/fs/fake-migrate.c"
    "${ISH_SOURCE_DIR}/fs/fake-rebuild.c"
)
target_include_directories(ish_fakefs PRIVATE
    "${ISH_SOURCE_DIR}"
    "${ISH_SOURCE_DIR}/deps/sqlite"
)
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
target_include_directories(ish_kernel PRIVATE
    "${ISH_SOURCE_DIR}"
    "${ISH_SOURCE_DIR}/deps/sqlite"
    "${ISH_CORE_BUILD_DIR}"
)
target_compile_definitions(ish_kernel PRIVATE
    ENGINE_ASBESTOS=1
    LOG_HANDLER_DPRINTF=1
)
target_compile_options(ish_kernel PRIVATE
    -Wno-switch
    -Wno-unused-parameter
    -Wno-sign-compare
    -Wno-pointer-sign
    -Wno-implicit-fallthrough
)
target_link_libraries(ish_kernel PUBLIC ish_emu ish_fakefs ish_sqlite Threads::Threads)

add_library(ish_core_session STATIC
    "${ISH_SOURCE_DIR}/app/core/CoreSession.c"
)
target_include_directories(ish_core_session PUBLIC
    "${ISH_SOURCE_DIR}/app/core"
    "${ISH_SOURCE_DIR}"
    "${ISH_SOURCE_DIR}/deps/sqlite"
    "${ISH_CORE_BUILD_DIR}"
)
target_compile_definitions(ish_core_session PRIVATE
    ENGINE_ASBESTOS=1
    LOG_HANDLER_DPRINTF=1
)
target_compile_options(ish_core_session PRIVATE
    -Wno-switch
    -Wno-unused-parameter
    -Wno-sign-compare
    -Wno-pointer-sign
    -Wno-implicit-fallthrough
)
target_link_libraries(ish_core_session PUBLIC ish_kernel ish_emu ish_fakefs ish_sqlite Threads::Threads ZLIB::ZLIB)

set(ISH_ASSET_QRC "${CMAKE_CURRENT_SOURCE_DIR}/assets/assets.qrc")
set(ISH_ASSET_CPP "${CMAKE_CURRENT_BINARY_DIR}/qrc_assets.cpp")
set(ISH_HOST_RCC "${QT_HOST_PATH}/libexec/rcc")
if(NOT EXISTS "${ISH_HOST_RCC}")
    set(ISH_HOST_RCC "${QT_HOST_PATH}/bin/rcc")
endif()
if(NOT EXISTS "${ISH_HOST_RCC}")
    get_filename_component(_qt_prefix "${Qt6Core_DIR}/../../.." ABSOLUTE)
    set(ISH_HOST_RCC "${_qt_prefix}/libexec/rcc")
endif()
if(NOT EXISTS "${ISH_HOST_RCC}")
    message(FATAL_ERROR "Unable to locate Qt resource compiler (rcc)")
endif()
add_custom_command(
    OUTPUT "${ISH_ASSET_CPP}"
    COMMAND "${ISH_HOST_RCC}" --no-zstd --name assets --output "${ISH_ASSET_CPP}" "${ISH_ASSET_QRC}"
    DEPENDS "${ISH_ASSET_QRC}"
    VERBATIM
)
set_source_files_properties("${ISH_ASSET_CPP}" PROPERTIES GENERATED TRUE)

qt_add_executable(ish_qt_android
    src/main.cpp
    src/IshSession.cpp
    src/IshSession.h
    src/RootfsManager.cpp
    src/RootfsManager.h
    src/ThemeManager.cpp
    src/ThemeManager.h
    src/UserPreferencesQt.cpp
    src/UserPreferencesQt.h
    src/WebSocketTransport.cpp
    src/WebSocketTransport.h
    src/WebChannelServer.cpp
    src/WebChannelServer.h
    src/PlatformServicesQt.cpp
    src/PlatformServicesQt.h
    src/FontCatalog.cpp
    src/FontCatalog.h
    src/RootModel.cpp
    src/RootModel.h
    src/RootFilesModel.cpp
    src/RootFilesModel.h
    src/RootfsUpgradeController.cpp
    src/RootfsUpgradeController.h
)
target_sources(ish_qt_android PRIVATE "${ISH_ASSET_CPP}")
target_include_directories(ish_qt_android PRIVATE
    "${ISH_SOURCE_DIR}/app/core"
    "${ISH_SOURCE_DIR}"
)
target_link_libraries(ish_qt_android PRIVATE
    Qt6::Core
    Qt6::Gui
    Qt6::Quick
    Qt6::WebView
    Qt6::WebChannel
    Qt6::WebSockets
    ZLIB::ZLIB
    ish_core_session
)

if(ANDROID AND NOT ANDROID_SDK_ROOT)
    set(ANDROID_SDK_ROOT "$ENV{ANDROID_SDK_ROOT}" CACHE PATH "Android SDK root")
endif()

set_target_properties(ish_qt_android PROPERTIES
    QT_ANDROID_APP_NAME "iSH Qt"
    QT_ANDROID_PACKAGE_NAME "com.mostafa637.ishqt"
    QT_ANDROID_VERSION_CODE 1
    QT_ANDROID_VERSION_NAME "1.0"
    QT_ANDROID_MIN_SDK_VERSION 28
    QT_ANDROID_TARGET_SDK_VERSION 36
    QT_ANDROID_COMPILE_SDK_VERSION 36
    QT_ANDROID_SDK_BUILD_TOOLS_REVISION "36.0.0"
)

set_source_files_properties(
    qml/IshIOSStyle/IOSPalette.qml
    PROPERTIES QT_QML_SINGLETON_TYPE TRUE
)

qt_add_qml_module(ish_qt_android
    URI IshQt
    VERSION 1.0
    QML_FILES
        qml/Main.qml
        qml/TerminalWeb.qml
        qml/TerminalDesktop.qml
        qml/TerminalScrollBar.qml
        qml/ArrowAccessoryButton.qml
        qml/ArrowAccessoryIOSButton.qml
        qml/AboutPage.qml
        qml/ThemesPage.qml
        qml/RootsPage.qml
        qml/FileBrowserPage.qml
        qml/PaletteEditor.qml
        qml/ThemeEditor.qml
        qml/FontPickerPage.qml
        qml/ExternalKeyboardPage.qml
        qml/AppearancePage.qml
        qml/ErrorDialog.qml
        qml/IshIOSStyle/IOSPalette.qml
        qml/IshIOSStyle/IOSButton.qml
        qml/IshIOSStyle/IOSToolButton.qml
        qml/IshIOSStyle/IOSToolBar.qml
        qml/IshIOSStyle/IOSTabBar.qml
        qml/IshIOSStyle/IOSTabButton.qml
        qml/IshIOSStyle/IOSDrawer.qml
        qml/IshIOSStyle/IOSPopup.qml
        qml/IshIOSStyle/IOSScrollBar.qml
        qml/IshIOSStyle/IOSPage.qml
        qml/IshIOSStyle/IOSItemDelegate.qml
        qml/IshIOSStyle/IOSLabel.qml
        qml/IshIOSStyle/IOSTextField.qml
        qml/IshIOSStyle/IOSTextArea.qml
        qml/IshIOSStyle/IOSComboBox.qml
        qml/IshIOSStyle/IOSCheckBox.qml
        qml/IshIOSStyle/IOSSwitch.qml
        qml/IshIOSStyle/IOSSlider.qml
        qml/IshIOSStyle/IOSProgressBar.qml
        qml/IshIOSStyle/IOSSpinBox.qml
)

# Qt 6.11.1 finalizes qt_add_executable automatically and creates the APK target.
