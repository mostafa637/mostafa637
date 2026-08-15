import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    signal editRequested(string themeName)

    header: IOSToolBar {
        title: "Themes"
        onBackClicked: root.closeRequested()
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 10

        IOSLabel {
            text: "Installed themes"
            font.pixelSize: 22
            font.bold: true
        }

        ListView {
            Layout.fillWidth: true
            Layout.fillHeight: true
            model: themes.themeNames

            delegate: IOSItemDelegate {
                width: ListView.view.width
                text: modelData
                onClicked: {
                    preferences.themeName = modelData
                    root.editRequested(modelData)
                }
            }
        }
    }
}
