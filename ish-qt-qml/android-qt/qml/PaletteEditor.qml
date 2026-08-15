import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    property string themeName: ""

    header: IOSToolBar {
        title: "Palette"
        onBackClicked: root.closeRequested()
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 10

        IOSLabel {
            text: "Palette editor"
            font.pixelSize: 22
            font.bold: true
        }

        IOSTextField {
            Layout.fillWidth: true
            placeholderText: "Background color"
        }

        IOSTextField {
            Layout.fillWidth: true
            placeholderText: "Foreground color"
        }
    }
}
