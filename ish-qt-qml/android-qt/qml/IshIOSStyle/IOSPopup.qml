import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Popup {
    id: popup
    property color styleWindowColor: "#f2f2f7"
    property string title: ""
    property string message: ""
    signal accepted()
    signal rejected()
    padding: 18
    modal: true
    dim: true
    focus: true
    closePolicy: Controls.Popup.CloseOnEscape
    anchors.centerIn: parent
    width: Math.min(parent ? parent.width - 32 : 360, 420)
    contentItem: Column {
        spacing: 12
        IOSLabel { text: popup.title; visible: popup.title.length > 0; font.bold: true; font.pixelSize: 19; width: parent.width; horizontalAlignment: Text.AlignHCenter }
        IOSLabel { text: popup.message; visible: popup.message.length > 0; width: parent.width; wrapMode: Text.WordWrap; horizontalAlignment: Text.AlignHCenter }
        Row {
            width: parent.width
            spacing: 8
            IOSButton { width: (parent.width - 8) / 2; text: "Cancel"; onClicked: { popup.rejected(); popup.close() } }
            IOSButton { width: (parent.width - 8) / 2; text: "OK"; onClicked: { popup.accepted(); popup.close() } }
        }
    }
    background: Rectangle { radius: 14; color: IOSPalette.elevatedSurface(popup.styleWindowColor); border.color: IOSPalette.separator(popup.styleWindowColor); border.width: 1 }
}
