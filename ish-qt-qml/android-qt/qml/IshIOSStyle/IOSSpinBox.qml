import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.SpinBox { id: control; property color styleWindowColor: "#f2f2f7"; implicitHeight: 40; contentItem: TextInput { text: control.textFromValue(control.value, control.locale); color: IOSPalette.text(control.styleWindowColor); horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter; readOnly: true }; background: Rectangle { radius: 9; color: IOSPalette.elevatedSurface(control.styleWindowColor); border.color: IOSPalette.separator(control.styleWindowColor); border.width: 1 } }
