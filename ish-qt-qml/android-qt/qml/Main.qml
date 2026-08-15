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
    property string editorThemeName: "Default"
    property string statusText: "Preparing iSH…"
    property bool pageReady: false
    property bool useWebTerminal: Qt.platform.os === "android"
    property bool controlModifier: false
    property bool sessionStopRequested: false
    property bool externalKeyboardActive: false
    property bool wideAccessory: width >= 700
    property int accessoryHeight: wideAccessory ? 43 : 36
    property int accessoryButtonSize: wideAccessory ? 43 : 32
    property int accessoryHorizontalPadding: wideAccessory ? 15 : 6
    property int accessoryVerticalPadding: wideAccessory ? 0 : 2
    property color accessoryBackground: isDarkColor(terminalBackground()) ? "#292929" : "#e5e5ea"
    property color accessoryForeground: isDarkColor(terminalBackground()) ? "#f5f5f7" : "#1c1c1e"

    ErrorDialog { id: errorDialog }

    function iconResource(name) {
        return "qrc:/ish-assets/ui/icons/" + name + (isDarkColor(window.terminalBackground()) ? "-dark.svg" : "-light.svg")
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
            const served = rootfsManager.terminalUrl(webChannel.url)
            if (served && String(served).length > 0)
                return served
        }
        return "qrc:/ish-assets/terminal/term.html"
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

    function hideKeyboardFromToolbar() {
        const item = terminalLoader.item
        if (item && item.hideKeyboard)
            item.hideKeyboard()
        else
            Qt.inputMethod.hide()
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
        if (Qt.platform.os !== "android") {
            window.externalKeyboardActive = false
            return
        }
        window.externalKeyboardActive = terminalLoader.item && terminalLoader.item.visible &&
                                        !Qt.inputMethod.visible
    }

    function closeSettings() {
        settingsVisible = false
        activePage = ""
        if (terminalLoader.item && terminalLoader.item.focusTerminal)
            terminalLoader.item.focusTerminal()
    }

    function openPage(pageName) {
        settingsVisible = false
        activePage = pageName
    }

    function closePage() {
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
                    else
                        ishSession.load()
                }
            }

            Connections {
                target: terminalLoader.item
                function onReady() {
                    window.pageReady = true
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
                color: window.terminalBackground()
                z: 5

                ColumnLayout {
                    anchors.fill: parent
                    spacing: 0

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.preferredHeight: 44
                        color: window.accessoryBackground

                        RowLayout {
                            anchors.fill: parent
                            anchors.leftMargin: 8
                            anchors.rightMargin: 8
                            spacing: 8

                            IOSButton {
                                text: "Back"
                                implicitWidth: 36
                                implicitHeight: 34
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
                                color: window.accessoryForeground
                                font.pixelSize: 17
                                font.bold: true
                                horizontalAlignment: Text.AlignHCenter
                            }

                            Item { implicitWidth: 36; implicitHeight: 34 }
                        }
                    }

                    Flickable {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        contentWidth: width
                        contentHeight: settingsColumn.implicitHeight + 32
                        clip: true

                        ColumnLayout {
                            id: settingsColumn
                            width: parent.width
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.margins: 16
                            spacing: 12

                            IOSLabel {
                                text: "Appearance"
                                font.pixelSize: 21
                                font.bold: true
                                color: preferences.terminalStyle.foregroundColor
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
                                color: preferences.terminalStyle.foregroundColor
                                opacity: 0.75
                            }

                            IOSLabel { text: "Theme"; color: preferences.terminalStyle.foregroundColor }
                            IOSComboBox {
                                Layout.fillWidth: true
                                model: themes.themeNames
                                currentIndex: Math.max(0, themes.themeNames.indexOf(preferences.themeName))
                                onActivated: preferences.themeName = currentText
                            }

                            IOSLabel { text: "Font family"; color: preferences.terminalStyle.foregroundColor }
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
                                    color: preferences.terminalStyle.foregroundColor
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
                                    color: preferences.terminalStyle.foregroundColor
                                }
                                IOSSwitch {
                                    checked: preferences.blinkCursor
                                    onToggled: preferences.blinkCursor = checked
                                }
                            }

                            IOSLabel { text: "Boot command"; color: preferences.terminalStyle.foregroundColor }
                            IOSTextField {
                                Layout.fillWidth: true
                                text: preferences.bootCommand.join(" ")
                                onEditingFinished: preferences.bootCommand = text.trim().split(/\s+/)
                            }

                            IOSLabel { text: "Launch command"; color: preferences.terminalStyle.foregroundColor }
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
                                color: preferences.terminalStyle.foregroundColor
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
                                color: preferences.terminalStyle.foregroundColor
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
                        item.navigateRequested.connect(function(pageName) { window.activePage = pageName })
                    if (item.editRequested)
                        item.editRequested.connect(function(themeName) {
                            window.editorThemeName = themeName
                            window.activePage = "themeEditor"
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
                spacing: 10

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
                            width: Math.min(26, window.accessoryButtonSize * 0.58)
                            height: Math.min(26, window.accessoryButtonSize * 0.58)
                            anchors.centerIn: parent
                            z: 2
                            source: accessoryButton.iconName.length > 0 ? window.vectorIcon(accessoryButton.iconName) : ""
                            visible: accessoryButton.iconName.length > 0 && status === Image.Ready
                            fillMode: Image.PreserveAspectFit
                            asynchronous: false
                            smooth: true
                            sourceSize: Qt.size(Math.min(26, window.accessoryButtonSize * 0.58), Math.min(26, window.accessoryButtonSize * 0.58))
                            onStatusChanged: if (status === Image.Error) console.warn("iSH icon load failed:", source)
                        }
                        Image {
                            id: bitmapImage
                            width: Math.min(26, window.accessoryButtonSize * 0.58)
                            height: Math.min(26, window.accessoryButtonSize * 0.58)
                            anchors.centerIn: parent
                            z: 2
                            source: accessoryButton.bitmapIconName.length > 0 ? window.bitmapIcon(accessoryButton.bitmapIconName) : ""
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
                    spacing: 7

                    AccessoryButton {
                        text: "Tab"
                        iconName: "tab"
                        onClicked: window.sendAccessoryInput("\t")
                    }

                    AccessoryButton {
                        text: "Control"
                        iconName: "control"
                        checkable: true
                        checked: window.controlModifier
                        onClicked: window.setControlModifier(!window.controlModifier)
                    }

                    AccessoryButton {
                        text: "Escape"
                        iconName: "escape"
                        onClicked: window.sendAccessoryInput("\u001b")
                    }

                    ArrowAccessoryButton {
                        id: arrowButton
                        Layout.preferredWidth: window.accessoryButtonSize
                        Layout.preferredHeight: window.accessoryButtonSize
                        buttonSize: window.accessoryButtonSize
                        foreground: window.accessoryForeground
                        darkMode: window.isDarkColor(window.terminalBackground())
                        normalColor: window.isDarkColor(window.terminalBackground()) ? "#555555" : "#f2f2f7"
                        pressedColor: window.isDarkColor(window.terminalBackground()) ? "#707070" : "#b8b8bd"
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
                    spacing: 7

                    Item {
                        Layout.preferredWidth: window.accessoryButtonSize
                        Layout.preferredHeight: window.accessoryButtonSize
                        AccessoryButton {
                            anchors.fill: parent
                            text: "Settings"
                            iconName: "gear"
                            onClicked: window.settingsVisible = true
                        }
                        Rectangle {
                            width: 9
                            height: 9
                            radius: 4.5
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
                        text: "Hide Keyboard"
                        bitmapIconName: "hide-keyboard"
                        onClicked: window.hideKeyboardFromToolbar()
                    }
                }
            }
        }
    }
}
