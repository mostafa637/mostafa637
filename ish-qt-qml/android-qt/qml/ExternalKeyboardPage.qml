import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    header: IOSToolBar {
        title: "External keyboard"
        onBackClicked: root.closeRequested()
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 12

        IOSLabel {
            text: "Keyboard settings"
            font.pixelSize: 22
            font.bold: true
        }

        IOSCheckBox {
            text: "Hide extra keys when a hardware keyboard is connected"
            checked: preferences.hideExtraKeysWithExternalKeyboard
            onToggled: preferences.hideExtraKeysWithExternalKeyboard = checked
            Layout.fillWidth: true
        }
    }
}
