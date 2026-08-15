import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    header: IOSToolBar {
        title: "About iSH"
        onBackClicked: root.closeRequested()
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 14

        IOSLabel {
            text: "iSH Qt"
            font.pixelSize: 28
            font.bold: true
        }

        IOSLabel {
            text: "A Qt/QML port of iSH using the native core, Asbestos emulator, fakefs and SQLite."
            Layout.fillWidth: true
            wrapMode: Text.WordWrap
        }

        IOSLabel {
            text: "Qt 6.11.1\nUTF-8 terminal I/O\nLinux and Android"
            Layout.fillWidth: true
        }
    }
}
