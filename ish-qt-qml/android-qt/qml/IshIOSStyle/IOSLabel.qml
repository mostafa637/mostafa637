import QtQuick
import IshQt

Text {
    id: label
    property color styleWindowColor: "#f2f2f7"
    color: IOSPalette.text(styleWindowColor)
    font.pixelSize: IOSMetrics.rowLabelSize
    wrapMode: Text.Wrap
}
