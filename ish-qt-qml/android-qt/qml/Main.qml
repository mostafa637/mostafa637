import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import IshQt

ApplicationWindow {
    id: window
    visible: true
    width: 900
    height: 620
    minimumWidth: 320
    minimumHeight: 260
    title: "iSH"
    color: terminalBackground()

    property bool settingsVisible: false
    property string activePage: ""
    property var pageStack: []
    property string editorThemeName: "Default"
    property string statusText: "Preparing iSH…"
    property bool pageReady: false
    property bool useWebTerminal: Qt.platform.os === "android"
    property bool controlModifier: false
    property bool sessionStopRequested: false
    property bool externalKeyboardActive: false
    // Android uses the bundled iSH-style QML keyboard instead of the system IME.
    // Keep it visible by default, matching iSH iOS after the terminal becomes first responder.
    property bool virtualKeyboardVisible: Qt.platform.os === "android"
    property bool wideAccessory: width >= 700
    // Terminal.storyboard: 50pt accessory stack, 31pt buttons, 6pt outer inset.
    property int accessoryHeight: IOSMetrics.accessoryBarHeight
    property int accessoryButtonSize: IOSMetrics.accessoryButtonWidth
    property int accessoryHorizontalPadding: IOSMetrics.accessoryOuterInset
    property int accessoryVerticalPadding: 0
    // The iSH accessory bar follows the interface/keyboard appearance, not
    // the WebView terminal palette (the Android terminal page is always black).
    property bool accessoryDarkMode: preferences && preferences.colorScheme === 2
    property color accessoryBackground: accessoryDarkMode ? "#2c2c2e" : "#f2f2f7"
    property color accessoryForeground: accessoryDarkMode ? "#f5f5f7" : "#1c1c1e"

    ErrorDialog { id: errorDialog }

    function iconResource(name) {
        return "qrc:/ish-assets/ui/icons/" + name + (isDarkColor(window.terminalBackground()) ? "-dark.png" : "-light.png")
    }

    // Accessory buttons follow the bar surface, not the WebView terminal surface.
    // Android's terminal page is intentionally black even when the selected
    // application palette is light, so using terminalBackground here can make
    // light icons disappear on the light iOS-style accessory bar.
    function accessoryIcon(name) {
        return "qrc:/ish-assets/ui/icons/" + name + (isDarkColor(window.accessoryBackground) ? "-dark.png" : "-light.png")
    }

    // Kept as an alias for pages/components that still use the old name.
    function vectorIcon(name) { return iconResource(name) }
    function bitmapIcon(name) { return iconResource(name) }

    function terminalBackground() {
        const p = preferences ? preferences.terminalStyle : {}
        return (p && p.backgroundColor) ? p.backgroundColor : "#000000"
    }

    function isDarkColor(value) {
        const c = Qt.color(value)
        if (!c || !c.valid)
            return true
        return (0.299 * c.r + 0.587 * c.g + 0.114 * c.b) < 0.5
    }

    function configureSession() {
        ishSession.configure(rootfsManager.rootPath,
                             preferences.encodeCommand(preferences.bootCommand),
                             preferences.encodeCommand(preferences.launchCommand))
    }

    function terminalPageUrl() {
        if (webChannel && webChannel.url) {
            const served = rootfsManager.terminalUrl(webChannel.url, webChannel.pageUrl)
            if (served && String(served).length > 0)
                return served
        }
        return rootfsManager.terminalUrl("", "")
    }

    function loadTerminalPage() {
        if (!rootfsManager.prepared)
            return
        configureSession()
        terminalLoader.source = window.useWebTerminal ? "TerminalWeb.qml" : "TerminalDesktop.qml"
        statusText = window.useWebTerminal ? "Loading terminal…" : "Loading host terminal…"
        if (terminalLoader.item && window.useWebTerminal)
            terminalLoader.item.loadPage(window.terminalPageUrl())
    }

    function setControlModifier(active) {
        window.controlModifier = Boolean(active)
        if (terminalLoader.item && terminalLoader.item.setControlModifier)
            terminalLoader.item.setControlModifier(window.controlModifier)
        if (terminalLoader.item && terminalLoader.item.focusTerminal)
            terminalLoader.item.focusTerminal()
    }

    function sendAccessoryInput(value) {
        if (!value || value.length === 0)
            return
        const item = terminalLoader.item
        if (item && item.sendAccessoryInput) {
            item.sendAccessoryInput(value)
        } else if (ishSession.alive) {
            ishSession.sendInput(value.replace(/\n/g, "\r"))
        }
        // A toolbar key is an input event just like a hardware key. Consume
        // sticky Control after the event, matching TerminalView.insertText:.
        if (window.controlModifier)
            window.setControlModifier(false)
        if (item && item.focusTerminal)
            item.focusTerminal()
    }

    function pasteFromToolbar() {
        const item = terminalLoader.item
        if (item && item.paste) {
            item.paste()
            return
        }
        if (ishSession.alive) {
            const text = platformServices.pasteText()
            if (text && text.length > 0)
                ishSession.sendInput(text.replace(/\n/g, "\r"))
        }
    }

    function toggleVirtualKeyboard() {
        if (Qt.platform.os !== "android") {
            Qt.inputMethod.hide()
            return
        }
        window.virtualKeyboardVisible = !window.virtualKeyboardVisible
        // The WebView terminal is not an editable text control, but explicitly
        // hide the platform IME so the bundled QML keyboard owns the layout.
        Qt.inputMethod.hide()
        const item = terminalLoader.item
        if (item && item.focusTerminal)
            item.focusTerminal()
    }

    function startSession() {
        if (!rootfsManager.prepared || ishSession.alive)
            return
        configureSession()
        if (!ishSession.start(rootfsManager.rootPath,
                              preferences.encodeCommand(preferences.bootCommand),
                              preferences.encodeCommand(preferences.launchCommand))) {
            platformServices.logDiagnostic("QML session", "Could not start session")
            window.sessionStopRequested = true
            statusText = "Could not start session"
            return
        }
        statusText = "Starting iSH…"
    }

    function restartSession() {
        if (!rootfsManager.prepared) {
            statusText = "Rootfs is not ready"
            return
        }
        // Mark the intentional stop before calling stop(): stop() emits
        // aliveChanged(false) synchronously, and the handler must not schedule
        // a second restart while this restart is already in progress.
        window.sessionStopRequested = true
        if (ishSession.alive)
            ishSession.stop()
        window.sessionStopRequested = false
        configureSession()
        if (!ishSession.start(rootfsManager.rootPath,
                              preferences.encodeCommand(preferences.bootCommand),
                              preferences.encodeCommand(preferences.launchCommand))) {
            platformServices.logDiagnostic("QML session restart", "Could not start session")
            window.sessionStopRequested = true
            statusText = "Could not start session"
            return
        }
        statusText = "Starting iSH…"
    }

    function stopSession() {
        window.sessionStopRequested = true
        window.setControlModifier(false)
        ishSession.stop()
    }

    function updateExternalKeyboardState() {
        // Qt.inputMethod.visible describes the platform IME, not a hardware
        // keyboard. The bundled iSH keyboard is intentionally independent of
        // that flag, so do not hide the accessory bar merely because the
        // platform IME is absent.
        window.externalKeyboardActive = false
    }

    function closeSettings() {
        settingsVisible = false
        activePage = ""
        pageStack = []
        if (terminalLoader.item && terminalLoader.item.focusTerminal)
            terminalLoader.item.focusTerminal()
    }

    function openPage(pageName) {
        if (!pageName || pageName.length === 0)
            return
        if (activePage !== "")
            pageStack = pageStack.concat([activePage])
        else
            settingsVisible = false
        activePage = pageName
    }

    function closePage() {
        if (pageStack.length > 0) {
            activePage = pageStack[pageStack.length - 1]
            pageStack = pageStack.slice(0, pageStack.length - 1)
            return
        }
        activePage = ""
        settingsVisible = true
    }

    Component.onCompleted: {
        loadTerminalPage()
        Qt.callLater(updateExternalKeyboardState)
    }

    Timer {
        interval: 350
        repeat: true
        running: window.useWebTerminal && !window.settingsVisible
        onTriggered: window.updateExternalKeyboardState()
    }

    Shortcut { sequence: "Ctrl+="; onActivated: if (terminalLoader.item && terminalLoader.item.increaseFontSize) terminalLoader.item.increaseFontSize() }
    Shortcut { sequence: "Ctrl++"; onActivated: if (terminalLoader.item && terminalLoader.item.increaseFontSize) terminalLoader.item.increaseFontSize() }
    Shortcut { sequence: "Ctrl+-"; onActivated: if (terminalLoader.item && terminalLoader.item.decreaseFontSize) terminalLoader.item.decreaseFontSize() }
    Shortcut { sequence: "Ctrl+0"; onActivated: if (terminalLoader.item && terminalLoader.item.resetFontSize) terminalLoader.item.resetFontSize() }
    Shortcut { sequence: "Ctrl+Shift+K"; onActivated: if (terminalLoader.item && terminalLoader.item.clearScrollback) terminalLoader.item.clearScrollback() }
    Shortcut { sequence: "Ctrl+,"; onActivated: window.settingsVisible = true }

    Connections {
        target: platformServices
        function onErrorReported(title, message) {
            errorDialog.showError(title, message)
        }
    }

    Connections {
        target: themes
        function onThemeError(message) {
            errorDialog.showError("Theme error", message)
        }
    }

    Connections {
        target: rootfsUpgrade
        function onFailed(message) {
            errorDialog.showError("Rootfs error", message)
        }
    }

    Connections {
        target: rootfsManager
        function onPreparedChanged() { loadTerminalPage() }
        function onProgressChanged(percent, message) { window.statusText = message + " (" + percent + "%)" }
        function onPreparationError(message) {
            window.statusText = message
            errorDialog.showError("Rootfs error", message)
        }
    }

    Connections {
        target: ishSession
        function onAliveChanged(alive) {
            window.statusText = alive ? "iSH is running" : "Session ended"
            if (alive) {
                window.sessionStopRequested = false
                return
            }
            window.setControlModifier(false)
            if (!window.sessionStopRequested && rootfsManager.prepared)
                Qt.callLater(window.restartSession)
        }
        function onLoaded() { window.statusText = "iSH is running" }
        function onSessionError(message) {
            platformServices.logDiagnostic("QML session", message)
            window.sessionStopRequested = true
            window.setControlModifier(false)
            window.statusText = message
            errorDialog.showError("Session error", message)
        }
        function onControlModifierConsumedSignal() {
            window.setControlModifier(false)
        }
    }

    Connections {
        target: preferences
        function onStyleChanged() {
            window.color = window.terminalBackground()
        }
    }

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        Item {
            id: terminalArea
            Layout.fillWidth: true
            Layout.fillHeight: true

            Rectangle {
                anchors.fill: parent
                color: window.terminalBackground()
            }

            Loader {
                id: terminalLoader
                anchors.fill: parent
                visible: !window.settingsVisible
                onLoaded: {
                    if (!item)
                        return
                    item.terminalStyle = ishSession.currentStyle
                    if (window.useWebTerminal)
                        item.loadPage(window.terminalPageUrl())
                    else {
                        ishSession.load()
                        window.startSession()
                    }
                }
            }

            Connections {
                target: terminalLoader.item
                function onReady() {
                    window.pageReady = true
                    window.startSession()
                    window.statusText = window.useWebTerminal ? "Terminal ready" : "Host terminal ready"
                }
                function onFailed(message) {
                    window.statusText = message
                    errorDialog.showError("Terminal error", message)
                }
            }

            Rectangle {
                id: settingsPanel
                anchors.fill: parent
                visible: window.settingsVisible && window.activePage === ""
                color: IOSPalette.surface(window.terminalBackground())
                z: 5

                ColumnLayout {
                    anchors.fill: parent
                    spacing: 0

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.preferredHeight: IOSMetrics.navigationBarHeight
                        color: IOSPalette.elevatedSurface(window.terminalBackground())

                        RowLayout {
                            anchors.fill: parent
                            anchors.leftMargin: IOSMetrics.accessoryOuterInset
                            anchors.rightMargin: IOSMetrics.accessoryOuterInset
                            spacing: IOSMetrics.accessorySpacing

                            IOSButton {
                                text: ""
                                implicitWidth: IOSMetrics.minimumTouchTarget
                                implicitHeight: IOSMetrics.minimumTouchTarget
                                onClicked: window.closeSettings()
                                contentItem: Image {
                                    source: window.bitmapIcon("xmark")
                                    fillMode: Image.PreserveAspectFit
                                    smooth: true
                                    sourceSize.width: 20
                                    sourceSize.height: 20
                                }
                                background: Rectangle { color: "transparent" }
                            }

                            IOSLabel {
                                Layout.fillWidth: true
                                text: "iSH Settings"
                                color: IOSPalette.text(window.terminalBackground())
                                font.pixelSize: IOSMetrics.navigationTitleSize
                                font.bold: true
                                horizontalAlignment: Text.AlignHCenter
                            }

                            Item { implicitWidth: IOSMetrics.minimumTouchTarget; implicitHeight: IOSMetrics.minimumTouchTarget }
                        }
                    }

                    Flickable {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        contentWidth: width
                        contentHeight: settingsColumn.implicitHeight + 36
                        clip: true

                        ColumnLayout {
                            id: settingsColumn
                            width: parent.width - 2 * IOSMetrics.sideInset(parent.width)
                            anchors.horizontalCenter: parent.horizontalCenter
                            y: 18
                            spacing: 0

                            IOSLabel {
                                Layout.fillWidth: true
                                Layout.preferredHeight: IOSMetrics.sectionHeaderHeight
                                text: "Appearance"
                                font.pixelSize: IOSMetrics.sectionLabelSize
                                font.bold: false
                                color: IOSPalette.secondaryText(preferences.terminalStyle.backgroundColor)
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: "Appearance"
                                onClicked: window.openPage("appearance")
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: "Themes"
                                onClicked: window.openPage("themes")
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: "Roots"
                                onClicked: window.openPage("roots")
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: "Browse root files"
                                onClicked: window.openPage("files")
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: "About iSH"
                                onClicked: window.openPage("about")
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: rootfsUpgrade.busy ? "Reinstalling…" : "Reinstall bundled rootfs"
                                enabled: !rootfsUpgrade.busy
                                onClicked: rootfsUpgrade.reinstallBundledRootfs()
                            }

                            IOSProgressBar {
                                Layout.fillWidth: true
                                visible: rootfsUpgrade.busy || rootfsUpgrade.progress > 0
                                from: 0
                                to: 100
                                value: rootfsUpgrade.progress
                            }

                            IOSLabel {
                                Layout.fillWidth: true
                                visible: rootfsUpgrade.message.length > 0
                                text: rootfsUpgrade.message
                                wrapMode: Text.WordWrap
                                color: IOSPalette.text(window.terminalBackground())
                                opacity: 0.75
                            }

                            IOSLabel { text: "Theme"; color: IOSPalette.text(window.terminalBackground()) }
                            IOSComboBox {
                                Layout.fillWidth: true
                                model: themes.themeNames
                                currentIndex: Math.max(0, themes.themeNames.indexOf(preferences.themeName))
                                onActivated: preferences.themeName = currentText
                            }

                            IOSLabel { text: "Font family"; color: IOSPalette.text(window.terminalBackground()) }
                            IOSTextField {
                                Layout.fillWidth: true
                                text: preferences.fontFamily
                                onEditingFinished: preferences.fontFamily = text
                            }

                            RowLayout {
                                Layout.fillWidth: true
                                IOSLabel {
                                    Layout.fillWidth: true
                                    text: "Font size: " + preferences.fontSize
                                    color: IOSPalette.text(window.terminalBackground())
                                }
                                IOSSpinBox {
                                    from: 6
                                    to: 48
                                    value: preferences.fontSize
                                    onValueModified: preferences.fontSize = value
                                }
                            }

                            RowLayout {
                                Layout.fillWidth: true
                                IOSLabel {
                                    Layout.fillWidth: true
                                    text: "Blink cursor"
                                    color: IOSPalette.text(window.terminalBackground())
                                }
                                IOSSwitch {
                                    checked: preferences.blinkCursor
                                    onToggled: preferences.blinkCursor = checked
                                }
                            }

                            IOSLabel { text: "Boot command"; color: IOSPalette.text(window.terminalBackground()) }
                            IOSTextField {
                                Layout.fillWidth: true
                                text: preferences.bootCommand.join(" ")
                                onEditingFinished: preferences.bootCommand = text.trim().split(/\s+/)
                            }

                            IOSLabel { text: "Launch command"; color: IOSPalette.text(window.terminalBackground()) }
                            IOSTextField {
                                Layout.fillWidth: true
                                text: preferences.launchCommand.join(" ")
                                onEditingFinished: preferences.launchCommand = text.trim().split(/\s+/)
                            }

                            RowLayout {
                                Layout.fillWidth: true
                                IOSButton {
                                    Layout.fillWidth: true
                                    text: "Copy selection"
                                    onClicked: if (terminalLoader.item && terminalLoader.item.copySelection) terminalLoader.item.copySelection()
                                }
                                IOSButton {
                                    Layout.fillWidth: true
                                    text: "Clear scrollback"
                                    onClicked: if (terminalLoader.item && terminalLoader.item.clearScrollback) terminalLoader.item.clearScrollback()
                                }
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: ishSession.alive ? "Stop session" : "Start session"
                                onClicked: ishSession.alive ? window.stopSession() : window.restartSession()
                            }

                            IOSLabel {
                                Layout.fillWidth: true
                                text: !rootfsManager.repositoryManaged ? "Rootfs repository metadata is unmanaged" :
                                      (rootfsManager.repositoryUpdateRequired ? "Repository metadata update is available" : "Repository metadata is current")
                                color: IOSPalette.text(window.terminalBackground())
                                opacity: 0.7
                                wrapMode: Text.WordWrap
                            }

                            IOSButton {
                                Layout.fillWidth: true
                                text: "Reset installed rootfs"
                                onClicked: {
                                    ishSession.stop()
                                    rootfsManager.resetInstalledData()
                                    rootfsManager.prepare()
                                    window.closeSettings()
                                }
                            }

                            IOSLabel {
                                Layout.fillWidth: true
                                wrapMode: Text.WordWrap
                                text: window.statusText
                                color: IOSPalette.text(window.terminalBackground())
                                opacity: 0.7
                            }
                        }
                    }
                }
            }

            Loader {
                id: subPageLoader
                anchors.fill: parent
                visible: window.activePage !== ""
                z: 10
                                    source: window.activePage === "about" ? "AboutPage.qml" :
                        (window.activePage === "themes" ? "ThemesPage.qml" :
                         (window.activePage === "roots" ? "RootsPage.qml" :
                          (window.activePage === "files" ? "FileBrowserPage.qml" :
                           (window.activePage === "appearance" ? "AppearancePage.qml" :
                           (window.activePage === "font" ? "FontPickerPage.qml" :
                            (window.activePage === "keyboard" ? "ExternalKeyboardPage.qml" :
                            (window.activePage === "themeEditor" ? "ThemeEditor.qml" : "")))))))

                onLoaded: {
                    if (!item)
                        return
                    item.pageBackground = window.terminalBackground()
                    item.pageForeground = preferences.terminalStyle.foregroundColor
                    item.closeRequested.connect(window.closePage)
                    if (item.navigateRequested)
                        item.navigateRequested.connect(function(pageName) { window.openPage(pageName) })
                    if (item.editRequested)
                        item.editRequested.connect(function(themeName) {
                            window.editorThemeName = themeName
                            window.openPage("themeEditor")
                        })
                    if (item.bootRootRequested)
                        item.bootRootRequested.connect(function() { window.restartSession() })
                    if (window.activePage === "themeEditor") {
                        item.originalName = window.editorThemeName
                        if (item.loadTheme)
                            item.loadTheme()
                    }
                }
            }
        }

        Rectangle {
            id: accessoryBar
            Layout.fillWidth: true
            Layout.preferredHeight: window.accessoryHeight
            visible: !window.settingsVisible &&
                     !(preferences.hideExtraKeysWithExternalKeyboard && window.externalKeyboardActive)
            color: window.accessoryBackground

                            RowLayout {
                                anchors.fill: parent
                                anchors.leftMargin: window.accessoryHorizontalPadding
                                anchors.rightMargin: window.accessoryHorizontalPadding
                                anchors.topMargin: window.accessoryVerticalPadding
                                anchors.bottomMargin: window.accessoryVerticalPadding
                                spacing: IOSMetrics.accessorySpacing

                component AccessoryButton: IOSButton {
                    id: accessoryButton
                    property string iconName: ""
                    property string bitmapIconName: ""
                    property string fallbackText: text
                    implicitWidth: window.accessoryButtonSize
                    implicitHeight: window.accessoryButtonSize
                    checkable: false
                    hoverEnabled: true
                    padding: 0
                    leftPadding: 0
                    rightPadding: 0
                    topPadding: 0
                    bottomPadding: 0

                    contentItem: Item {
                        Image {
                            id: vectorImage
                            width: Math.min(24, window.accessoryButtonSize * 0.72)
                            height: Math.min(24, window.accessoryButtonSize * 0.72)
                            anchors.centerIn: parent
                            z: 2
                            source: accessoryButton.iconName.length > 0 ? window.accessoryIcon(accessoryButton.iconName) : ""
                            visible: accessoryButton.iconName.length > 0 && status === Image.Ready
                            fillMode: Image.PreserveAspectFit
                            asynchronous: false
                            smooth: true
                            sourceSize: Qt.size(Math.min(26, window.accessoryButtonSize * 0.58), Math.min(26, window.accessoryButtonSize * 0.58))
                            onStatusChanged: if (status === Image.Error) console.warn("iSH icon load failed:", source)
                        }
                        Image {
                            id: bitmapImage
                            width: Math.min(24, window.accessoryButtonSize * 0.72)
                            height: Math.min(24, window.accessoryButtonSize * 0.72)
                            anchors.centerIn: parent
                            z: 2
                            source: accessoryButton.bitmapIconName.length > 0 ? window.accessoryIcon(accessoryButton.bitmapIconName) : ""
                            visible: accessoryButton.bitmapIconName.length > 0 && status === Image.Ready
                            fillMode: Image.PreserveAspectFit
                            asynchronous: false
                            smooth: true
                            sourceSize: Qt.size(Math.min(26, window.accessoryButtonSize * 0.58), Math.min(26, window.accessoryButtonSize * 0.58))
                            onStatusChanged: if (status === Image.Error) console.warn("iSH bitmap load failed:", source)
                        }
                        Text {
                            anchors.fill: parent
                            z: 1
                            visible: !vectorImage.visible && !bitmapImage.visible
                            text: accessoryButton.fallbackText
                            color: window.accessoryForeground
                            font.pixelSize: 15
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter
                        }
                    }
                    background: Rectangle {
                        radius: 9
                        color: accessoryButton.pressed || accessoryButton.checked
                               ? (window.isDarkColor(window.terminalBackground()) ? "#666666" : "#b8b8bd")
                               : (window.isDarkColor(window.terminalBackground()) ? "#555555" : "#f2f2f7")
                        border.width: 1
                        border.color: window.isDarkColor(window.terminalBackground()) ? "#707070" : "#d1d1d6"
                        opacity: accessoryButton.enabled ? 1.0 : 0.55
                    }
                }

                // iSH's original left cluster: Tab, Control, Escape, then one
                // compound ArrowBarButton. Keeping it as a group preserves the
                // visual rhythm and touch target sizes of the iOS accessory bar.
                    RowLayout {
                    id: primaryKeyGroup
                    Layout.alignment: Qt.AlignVCenter
                    spacing: IOSMetrics.accessorySpacing

                    AccessoryButton {
                        text: "Tab"
                        iconName: "tab"
                        fallbackText: "Tab"
                        onClicked: window.sendAccessoryInput("\t")
                    }

                    AccessoryButton {
                        text: "Control"
                        iconName: "control"
                        fallbackText: "Ctrl"
                        checkable: true
                        checked: window.controlModifier
                        onClicked: window.setControlModifier(!window.controlModifier)
                    }

                    AccessoryButton {
                        text: "Escape"
                        iconName: "escape"
                        fallbackText: "Esc"
                        onClicked: window.sendAccessoryInput("\u001b")
                    }

                    ArrowAccessoryButton {
                        id: arrowButton
                        Layout.preferredWidth: window.accessoryButtonSize
                        Layout.preferredHeight: window.accessoryButtonSize
                        buttonSize: window.accessoryButtonSize
                        foreground: window.accessoryForeground
                        darkMode: window.isDarkColor(window.accessoryBackground)
                        normalColor: window.accessoryDarkMode ? "#555555" : "#ffffff"
                        pressedColor: window.accessoryDarkMode ? "#707070" : "#d1d1d6"
                        onDirectionRequested: function(escapeSequence) { window.sendAccessoryInput(escapeSequence) }
                    }
                }

                // This flexible space is intentional: on iPhone it keeps the
                // arrow cluster and the utility cluster visually separated, and
                // on desktop it scales without stretching individual buttons.
                Item { Layout.fillWidth: true; Layout.minimumWidth: 8 }

                    RowLayout {
                    id: utilityKeyGroup
                    Layout.alignment: Qt.AlignVCenter
                    spacing: IOSMetrics.accessorySpacing

                    AccessoryButton {
                        text: "Back"
                        iconName: "arrow-left"
                        visible: window.activePage !== ""
                        enabled: window.activePage !== ""
                        onClicked: if (window.activePage !== "") window.closePage()
                    }

                    Item {
                        Layout.preferredWidth: window.accessoryButtonSize
                        Layout.preferredHeight: window.accessoryButtonSize
                        AccessoryButton {
                            anchors.fill: parent
                            text: "Settings"
                            // The iOS storyboard uses UIButtonTypeInfoLight here;
                            // use a bundled rasterized copy so Android font coverage
                            // cannot replace the original glyph with a missing symbol.
                            iconName: "info"
                            fallbackText: "Info"
                            onClicked: window.settingsVisible = true
                        }
                        Rectangle {
                                width: 12
                                height: 12
                                radius: 6
                            anchors.right: parent.right
                            anchors.top: parent.top
                            anchors.rightMargin: -1
                            anchors.topMargin: -1
                            color: "#ff3b30"
                            visible: rootfsManager.repositoryUpdateRequired
                            border.width: 1
                            border.color: window.accessoryBackground
                        }
                    }

                    AccessoryButton {
                        text: "Paste"
                        bitmapIconName: "paste"
                        onClicked: window.pasteFromToolbar()
                    }

                    AccessoryButton {
                        text: window.virtualKeyboardVisible ? "Hide Keyboard" : "Keyboard"
                        bitmapIconName: "hide-keyboard"
                        onClicked: window.toggleVirtualKeyboard()
                    }
                }
            }
        }

        IshVirtualKeyboard {
            id: virtualKeyboard
            Layout.fillWidth: true
            Layout.preferredHeight: visible
                                         ? Math.min(310, Math.max(226, window.width * 0.58))
                                         : 0
            visible: window.useWebTerminal && window.virtualKeyboardVisible && !window.settingsVisible
            darkMode: window.accessoryDarkMode
            onInputRequested: function(value) { window.sendAccessoryInput(value) }
        }
    }
}
