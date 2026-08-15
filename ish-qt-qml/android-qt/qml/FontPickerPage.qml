import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Font family"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: "Current font"; font.pixelSize: 22; font.bold: true }; IOSTextField { Layout.fillWidth: true; text: preferences.fontFamily; onEditingFinished: preferences.fontFamily = text }; IOSLabel { text: "Noto Sans Mono is bundled as the default font."; wrapMode: Text.WordWrap; Layout.fillWidth: true } }
}
