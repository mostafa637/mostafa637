import QtQuick
import QtQuick.Controls as Controls

Controls.Dialog {
    id: dialog
    property string message: ""
    title: "Error"
    modal: true
    // Give Fusion a concrete width so contentItem sizing cannot recurse
    // through Dialog.implicitWidth when the error text is wrapped.
    implicitWidth: 360
    standardButtons: Controls.Dialog.Ok
    function showError(titleText, messageText) { title = titleText; message = messageText; open() }
    contentItem: Text { text: dialog.message; wrapMode: Text.Wrap; padding: 16 }
}
