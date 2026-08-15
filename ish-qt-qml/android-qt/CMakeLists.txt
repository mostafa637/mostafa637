cmake_minimum_required(VERSION 3.22)

project(ish_qt_android VERSION 1.0 LANGUAGES C CXX ASM)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

find_package(Qt6 6.11.1 REQUIRED COMPONENTS Core Gui Quick Svg WebView WebChannel WebSockets)
qt_standard_project_setup(REQUIRES 6.8)

set(ISH_ROOT "${CMAKE_CURRENT_LIST_DIR}/..")

# Build the same direct iSH core session on Android and Linux. Android uses
# the existing ABI-aware core CMake; Linux links the validated host headless
# archives and compiles only the platform-neutral session/bootstrap layer.
if(ANDROID)
    add_subdirectory(
        "${ISH_ROOT}/android/app/src/main/cpp"
        "${CMAKE_CURRENT_BINARY_DIR}/ish-core-build"
    )
    set(ISH_SESSION_TARGET ish_core_session)
else()
    find_package(Threads REQUIRED)
    set(ISH_HEADLESS_BUILD "${ISH_ROOT}/build-headless")
    foreach(_archive libish.a libish_emu.a libfakefs.a)
        if(NOT EXISTS "${ISH_HEADLESS_BUILD}/${_archive}")
            message(FATAL_ERROR "Missing validated host core archive: ${ISH_HEADLESS_BUILD}/${_archive}")
        endif()
    endforeach()

    add_library(ish_headless_core STATIC IMPORTED GLOBAL)
    set_target_properties(ish_headless_core PROPERTIES
        IMPORTED_LOCATION "${ISH_HEADLESS_BUILD}/libish.a")
    add_library(ish_headless_emu STATIC IMPORTED GLOBAL)
    set_target_properties(ish_headless_emu PROPERTIES
        IMPORTED_LOCATION "${ISH_HEADLESS_BUILD}/libish_emu.a")
    add_library(ish_headless_fakefs STATIC IMPORTED GLOBAL)
    set_target_properties(ish_headless_fakefs PROPERTIES
        IMPORTED_LOCATION "${ISH_HEADLESS_BUILD}/libfakefs.a")

    add_library(ish_core_session STATIC
        "${ISH_ROOT}/app/core/CoreSession.c"
        "${ISH_ROOT}/app/core/CoreFilesystem.c"
        "${ISH_ROOT}/app/core/CoreFilesystem.h"
        "${ISH_ROOT}/app/core/CoreClipboard.c"
        "${ISH_ROOT}/app/core/CoreLocation.c"
    )
    target_include_directories(ish_core_session PRIVATE
        "${ISH_ROOT}"
        "${ISH_ROOT}/deps/sqlite"
        "${ISH_ROOT}/app/core"
        "${ISH_HEADLESS_BUILD}"
    )
    target_compile_definitions(ish_core_session PRIVATE
        _GNU_SOURCE
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
    target_link_libraries(ish_core_session PRIVATE
        ish_headless_core
        ish_headless_emu
        ish_headless_fakefs
        Threads::Threads
        sqlite3
        m
        dl
    )
    set(ISH_SESSION_TARGET ish_core_session)
endif()

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
    "${ISH_ROOT}/app/core"
)

target_link_libraries(ish_qt_android PRIVATE
    Qt6::Core
    Qt6::Gui
    Qt6::Quick
    Qt6::Svg
    Qt6::WebView
    Qt6::WebChannel
    Qt6::WebSockets
    ${ISH_SESSION_TARGET}
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
