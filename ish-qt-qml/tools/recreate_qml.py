from pathlib import Path

ROOT = Path('/home/ubuntu/ish-android/android-qt/qml')

files = {
'IshIOSStyle/IOSPalette.qml': '''pragma Singleton
import QtQuick

QtObject {
    readonly property color blue: "#007aff"
    readonly property color green: "#34c759"
    readonly property color red: "#ff3b30"
    readonly property color orange: "#ff9500"
    readonly property color secondary: "#8e8e93"
    function isDark(background) { var c = Qt.color(background); return c.valid && (0.299*c.r + 0.587*c.g + 0.114*c.b) < 0.5 }
    function surface(background) { return isDark(background) ? "#1c1c1e" : "#f2f2f7" }
    function elevatedSurface(background) { return isDark(background) ? "#2c2c2e" : "#ffffff" }
    function text(background) { return isDark(background) ? "#f5f5f7" : "#1c1c1e" }
    function secondaryText(background) { return isDark(background) ? "#aeaeb2" : "#6c6c70" }
    function separator(background) { return isDark(background) ? "#48484a" : "#c6c6c8" }
}
''',
'IshIOSStyle/IOSButton.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Button {
    id: control
    property color styleWindowColor: Controls.ApplicationWindow.window ? Controls.ApplicationWindow.window.color : "#f2f2f7"
    implicitHeight: 42
    padding: 10
    font.pixelSize: 16
    contentItem: Text { text: control.text; color: control.enabled ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor); font: control.font; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { radius: 10; color: control.pressed ? IOSPalette.separator(control.styleWindowColor) : IOSPalette.elevatedSurface(control.styleWindowColor); border.width: 1; border.color: IOSPalette.separator(control.styleWindowColor); opacity: control.enabled ? 1 : .55 }
}
''',
'IshIOSStyle/IOSToolButton.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ToolButton {
    id: control
    property color styleWindowColor: Controls.ApplicationWindow.window ? Controls.ApplicationWindow.window.color : "#f2f2f7"
    implicitWidth: 42
    implicitHeight: 40
    font.pixelSize: 16
    contentItem: Text { text: control.text; color: control.enabled ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor); font: control.font; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { radius: 9; color: control.pressed ? IOSPalette.separator(control.styleWindowColor) : "transparent"; border.width: control.pressed ? 1 : 0; border.color: IOSPalette.separator(control.styleWindowColor) }
}
''',
'IshIOSStyle/IOSToolBar.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

Rectangle {
    id: bar
    property string title: ""
    property color styleWindowColor: parent && parent.pageBackground ? parent.pageBackground : "#f2f2f7"
    signal backClicked()
    default property alias toolItems: actions.data
    implicitHeight: 48
    color: IOSPalette.surface(styleWindowColor)
    border.color: IOSPalette.separator(styleWindowColor)
    border.width: 1
    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 6
        anchors.rightMargin: 6
        spacing: 4
        IOSToolButton { text: "‹  Back"; visible: bar.title.length > 0; onClicked: bar.backClicked() }
        IOSLabel { Layout.fillWidth: true; text: bar.title; font.bold: true; font.pixelSize: 17; horizontalAlignment: Text.AlignHCenter }
        RowLayout { id: actions; spacing: 2 }
    }
}
''',
'IshIOSStyle/IOSPage.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Page {
    id: page
    property color pageBackground: "#f2f2f7"
    property color pageForeground: "#1c1c1e"
    signal closeRequested()
    signal navigateRequested(string pageName)
    signal editRequested(string themeName)
    signal bootRootRequested()
    background: Rectangle { color: page.pageBackground }
}
''',
'IshIOSStyle/IOSLabel.qml': '''import QtQuick
import IshQt

Text {
    id: label
    property color styleWindowColor: "#f2f2f7"
    color: IOSPalette.text(styleWindowColor)
    font.pixelSize: 16
    wrapMode: Text.Wrap
}
''',
'IshIOSStyle/IOSItemDelegate.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ItemDelegate {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: 44
    contentItem: Text { text: control.text; color: IOSPalette.text(control.styleWindowColor); font: control.font; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { color: control.pressed || control.highlighted ? IOSPalette.separator(control.styleWindowColor) : "transparent"; radius: 8 }
}
''',
'IshIOSStyle/IOSTextField.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.TextField {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: 40
    color: IOSPalette.text(styleWindowColor)
    placeholderTextColor: IOSPalette.secondaryText(styleWindowColor)
    background: Rectangle { radius: 9; color: IOSPalette.elevatedSurface(control.styleWindowColor); border.width: control.activeFocus ? 2 : 1; border.color: control.activeFocus ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor) }
}
''',
'IshIOSStyle/IOSComboBox.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ComboBox {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: 40
    contentItem: Text { text: control.displayText; color: IOSPalette.text(control.styleWindowColor); font: control.font; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { radius: 9; color: IOSPalette.elevatedSurface(control.styleWindowColor); border.width: control.activeFocus ? 2 : 1; border.color: control.activeFocus ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor) }
}
''',
'IshIOSStyle/IOSCheckBox.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.CheckBox {
    id: control
    property color styleWindowColor: "#f2f2f7"
    contentItem: Text { text: control.text; color: IOSPalette.text(control.styleWindowColor); font: control.font; leftPadding: control.indicator.width + 8; verticalAlignment: Text.AlignVCenter }
    indicator: Rectangle { implicitWidth: 24; implicitHeight: 24; x: control.leftPadding; y: (control.height-height)/2; radius: 6; color: control.checked ? IOSPalette.blue : "transparent"; border.width: 1; border.color: control.checked ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor); Text { anchors.centerIn: parent; text: "✓"; color: "white"; visible: control.checked } }
}
''',
'IshIOSStyle/IOSSwitch.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Switch {
    id: control
    property color styleWindowColor: "#f2f2f7"
    indicator: Rectangle { implicitWidth: 50; implicitHeight: 30; x: control.leftPadding; y: (control.height-height)/2; radius: 15; color: control.checked ? IOSPalette.green : IOSPalette.separator(control.styleWindowColor); Rectangle { width: 26; height: 26; radius: 13; y: 2; x: control.checked ? parent.width-width-2 : 2; color: "white" } }
    contentItem: Text { text: control.text; color: IOSPalette.text(control.styleWindowColor); font: control.font; leftPadding: control.indicator.width + 10; verticalAlignment: Text.AlignVCenter }
}
''',
'IshIOSStyle/IOSSlider.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Slider { id: control; property color styleWindowColor: "#f2f2f7"; implicitHeight: 36; background: Rectangle { x: 0; y: control.height/2-2; width: control.width; height: 4; radius: 2; color: IOSPalette.separator(control.styleWindowColor) }; handle: Rectangle { x: control.visualPosition * (control.width-width); y: control.height/2-height/2; width: 22; height: 22; radius: 11; color: IOSPalette.blue } }
''',
'IshIOSStyle/IOSProgressBar.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ProgressBar { id: control; property color styleWindowColor: "#f2f2f7"; implicitHeight: 8; background: Rectangle { radius: 4; color: IOSPalette.separator(control.styleWindowColor) }; contentItem: Item { Rectangle { width: control.visualPosition * parent.width; height: parent.height; radius: 4; color: IOSPalette.blue } } }
''',
'IshIOSStyle/IOSSpinBox.qml': '''import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.SpinBox { id: control; property color styleWindowColor: "#f2f2f7"; implicitHeight: 40; contentItem: TextInput { text: control.textFromValue(control.value, control.locale); color: IOSPalette.text(control.styleWindowColor); horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter; readOnly: true }; background: Rectangle { radius: 9; color: IOSPalette.elevatedSurface(control.styleWindowColor); border.color: IOSPalette.separator(control.styleWindowColor); border.width: 1 } }
''',
'ErrorDialog.qml': '''import QtQuick
import QtQuick.Controls as Controls

Controls.Dialog {
    id: dialog
    property string message: ""
    title: "Error"
    modal: true
    standardButtons: Controls.Dialog.Ok
    function showError(titleText, messageText) { title = titleText; message = messageText; open() }
    contentItem: Text { text: dialog.message; wrapMode: Text.Wrap; padding: 16 }
}
''',
'AboutPage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "About iSH"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 20; spacing: 14; IOSLabel { text: "iSH Qt"; font.pixelSize: 28; font.bold: true }; IOSLabel { text: "A Qt/QML port of iSH using the native core, Asbestos emulator, fakefs and SQLite."; Layout.fillWidth: true; wrapMode: Text.WordWrap }; IOSLabel { text: "Qt 6.11.1\nUTF-8 terminal I/O\nLinux and Android"; Layout.fillWidth: true } }
}
''',
'AppearancePage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Appearance"; onBackClicked: root.closeRequested() }
    Flickable { anchors.fill: parent; contentWidth: width; contentHeight: column.implicitHeight + 32; clip: true; ColumnLayout { id: column; width: parent.width; anchors.margins: 16; spacing: 12; IOSLabel { text: "Terminal appearance"; font.pixelSize: 22; font.bold: true }; IOSLabel { text: "Theme" }; IOSComboBox { Layout.fillWidth: true; model: themes.themeNames; currentIndex: Math.max(0, themes.themeNames.indexOf(preferences.themeName)); onActivated: preferences.themeName = currentText }; IOSLabel { text: "Font size" }; IOSSlider { Layout.fillWidth: true; from: 6; to: 32; value: preferences.fontSize; onMoved: preferences.fontSize = Math.round(value) }; IOSCheckBox { text: "Blink cursor"; checked: preferences.blinkCursor; onToggled: preferences.blinkCursor = checked } } }
}
''',
'ThemesPage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    signal editRequested(string themeName)
    header: IOSToolBar { title: "Themes"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: "Installed themes"; font.pixelSize: 22; font.bold: true }; ListView { Layout.fillWidth: true; Layout.fillHeight: true; model: themes.themeNames; delegate: IOSItemDelegate { width: ListView.view.width; text: modelData; onClicked: { preferences.themeName = modelData; root.editRequested(modelData) } } } }
}
''',
'RootsPage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Root filesystems"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: rootfsManager.prepared ? "Rootfs is ready" : "Rootfs is being prepared"; font.pixelSize: 20 }; IOSLabel { text: rootfsManager.rootPath; Layout.fillWidth: true; wrapMode: Text.Wrap }; IOSButton { text: "Prepare rootfs"; enabled: !rootfsManager.prepared; onClicked: rootfsManager.prepare() }; IOSButton { text: "Boot session"; enabled: rootfsManager.prepared; onClicked: root.bootRootRequested() } }
}
''',
'FileBrowserPage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Root files"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: "Files in the installed rootfs"; font.pixelSize: 20 }; IOSLabel { text: rootfsManager.rootPath; Layout.fillWidth: true; wrapMode: Text.Wrap }; IOSLabel { text: "File browsing is available through the rootfs model when it is prepared."; Layout.fillWidth: true; wrapMode: Text.Wrap } }
}
''',
'ExternalKeyboardPage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "External keyboard"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 12; IOSLabel { text: "Keyboard settings"; font.pixelSize: 22; font.bold: true }; IOSCheckBox { text: "Hide extra keys when a hardware keyboard is connected"; checked: preferences.hideExtraKeysWithExternalKeyboard; onToggled: preferences.hideExtraKeysWithExternalKeyboard = checked; Layout.fillWidth: true } }
}
''',
'FontPickerPage.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Font family"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: "Current font"; font.pixelSize: 22; font.bold: true }; IOSTextField { Layout.fillWidth: true; text: preferences.fontFamily; onEditingFinished: preferences.fontFamily = text }; IOSLabel { text: "Noto Sans Mono is bundled as the default font."; wrapMode: Text.WordWrap; Layout.fillWidth: true } }
}
''',
'PaletteEditor.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    property string themeName: ""
    header: IOSToolBar { title: "Palette"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: "Palette editor"; font.pixelSize: 22; font.bold: true }; IOSTextField { Layout.fillWidth: true; placeholderText: "Background color" }; IOSTextField { Layout.fillWidth: true; placeholderText: "Foreground color" } }
}
''',
'ThemeEditor.qml': '''import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    property string originalName: ""
    function loadTheme() { }
    header: IOSToolBar { title: "Edit theme"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: root.originalName.length ? root.originalName : "Theme"; font.pixelSize: 22; font.bold: true }; IOSTextField { Layout.fillWidth: true; placeholderText: "Theme name" }; IOSButton { text: "Save"; onClicked: root.closeRequested() } }
}
''',
}

for rel, content in files.items():
    path = ROOT / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding='utf-8')
print(f'WROTE {len(files)} QML files')
