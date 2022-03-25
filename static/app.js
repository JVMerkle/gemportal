/* SPDX-FileCopyrightText: 2021-2022 Contributors of gemportal */
/* SPDX-License-Identifier: AGPL-3.0-only */

function onSubmit() {
    let url = document.getElementById("idUrl");

    if (url.value.startsWith("gemini://")) {
        url.value = url.value.slice(9)
    }

    const actionSrc = BASE_HREF + url.value;
    const insecure = document.getElementById("idInsecure");

    if (insecure.checked) {
        window.location.href = actionSrc + "?insecure=on";
    } else {
        window.location.href = actionSrc;
    }

    return false;
}

function onUnsafeCheck() {
    const text = `*WARNING*
    The insecure mode disables all TLS-based checks, such as:
    - checking for the correct hostname or IP in the certificate and
    - checking for expired certificates.

    Are you sure you want to enable the insecure mode?`;

    const insecure = document.getElementById("idInsecure");
    const checked = !insecure.checked; // Because we are in an 'onClick' hook
    if (!checked) {
        if (!confirm(text)) {
            return false; // Reject checking the checkbox
        }
    }

    return true;
}