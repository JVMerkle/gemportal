function onSubmit() {
    var url = document.getElementById("idUrl");
    var actionSrc = BASE_HREF + url.value;
    var unsafe = document.getElementById("idUnsafe");

    if(unsafe.checked) {
        window.location.href = actionSrc + "?unsafe=on";
    } else {
        window.location.href = actionSrc;
    }

    return false;
}