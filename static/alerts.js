var currentPhone = ""

async function loadAlerts() {
    var content = document.getElementById("alertsContent")
    var alertsList = document.getElementById("alertsList")

    try {
        var response = await fetch("/api/alerts")
        if (!response.ok) {
            throw new Error("Server error: " + response.status)
        }
        var allAlerts = await response.json()
        if (!allAlerts) allAlerts = []

        var alerts = []
        for (var i = 0; i < allAlerts.length; i++) {
            if (allAlerts[i].phone === currentPhone) {
                alerts.push(allAlerts[i])
            }
        }

        alertsList.style.display = "block"

        if (alerts.length === 0) {
            content.innerHTML = '<div class="empty"><div class="empty-icon">🔔</div>No alerts found for this number.<br><a href="/' + METRO + '" class="back-link">Browse tee times</a> and select a course to create one.</div>'
            return
        }

        var html = '<div class="alerts-list">'
        for (var i = 0; i < alerts.length; i++) {
            var a = alerts[i]
            var statusClass = a.active ? "alert-active" : "alert-inactive"
            var statusText = a.active ? "Active" : "Triggered"

            html += '<div class="alert-item">'
            html += '  <div class="alert-info">'
            html += '    <div class="alert-course">' + a.course + '</div>'
            html += '    <div class="alert-details">' + a.date + ' · ' + a.startTime + ' – ' + a.endTime + '</div>'
            html += '    <div class="alert-meta">Created ' + a.createdAt + '</div>'
            html += '  </div>'
            html += '  <div class="alert-actions">'
            html += '    <span class="alert-status ' + statusClass + '">' + statusText + '</span>'
            html += '    <button class="btn-delete" onclick="removeAlert(\'' + a.id + '\')">Delete</button>'
            html += '  </div>'
            html += '</div>'
        }
        html += '</div>'

        content.innerHTML = html
    } catch (err) {
        alertsList.style.display = "block"
        content.innerHTML = '<div class="empty"><div class="empty-icon">⚠️</div>Failed to load alerts.</div>'
    }
}

function lookupAlerts() {
    var phone = document.getElementById("phone").value
    if (!phone) return

    currentPhone = phone
    loadAlerts()
}

async function removeAlert(id) {
    try {
        var response = await fetch("/api/alerts/delete?id=" + id, {
            method: "POST"
        })

        if (!response.ok) {
            throw new Error("Server error: " + response.status)
        }

        loadAlerts()
    } catch (err) {
        alert("Failed to delete alert.")
    }
}

document.getElementById("lookupBtn").addEventListener("click", lookupAlerts)
document.getElementById("phone").addEventListener("keydown", function(e) {
    if (e.key === "Enter") lookupAlerts()
})
