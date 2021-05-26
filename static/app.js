function onSubmit() {
    var url = document.getElementById("idUrl");

    if(url.value.startsWith("gemini://")) {
        url.value = url.value.slice(9)
    }

    var actionSrc = BASE_HREF + url.value;
    var unsafe = document.getElementById("idUnsafe");

    if(unsafe.checked) {
        window.location.href = actionSrc + "?unsafe=on";
    } else {
        window.location.href = actionSrc;
    }

    return false;
}