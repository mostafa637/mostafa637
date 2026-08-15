import QtQuick
import IshQt

Text {
    id: label
    property color styleWindowColor: "#f2f2f7"
    color: IOSPalette.text(styleWindowColor)
    font.pixelSize: 16
    wrapMode: Text.Wrap
}
