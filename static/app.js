var allTimes = []

function getBaseCourse(name) {
    if (name.indexOf("Kennedy") === 0) return "Kennedy"
    if (name.indexOf("Fox Hollow") === 0) return "Fox Hollow"
    if (name.indexOf("Homestead") === 0) return "Homestead"
    if (name.indexOf("Harvard Gulch") === 0) return "Harvard Gulch"
    if (name.indexOf("South Suburban") === 0) return "South Suburban"
    if (name.indexOf("Foothills") === 0) return "Foothills"
    if (name.indexOf("Meadows") === 0) return "Meadows"
    if (name.indexOf("Broken Tee") === 0) return "Broken Tee"
    if (name.indexOf("Fossil Trace") === 0) return "Fossil Trace"
    return name.replace(" Back Nine", "")
}

async function fetchTimes() {
    var date = document.getElementById("date").value
    document.getElementById("results").innerHTML = '<div class="loading"><span class="loading-spinner"></span>Loading tee times...</div>'
    document.getElementById("count").textContent = ""

    try {
        var response = await fetch("/teetimes?date=" + date)
        if (!response.ok) {
            throw new Error("Server error: " + response.status)
        }
        allTimes = await response.json()
        if (!allTimes) allTimes = []
    } catch (err) {
        document.getElementById("results").innerHTML = '<div class="empty"><div class="empty-icon">⚠️</div>Failed to load tee times. Please try again.</div>'
        document.getElementById("count").textContent = ""
        return
    }

    updateCourseFilter()
    displayTimes()
}

function updateCourseFilter() {
    var courseSelect = document.getElementById("course")
    var currentValue = courseSelect.value
    var courses = []

    for (var i = 0; i < allTimes.length; i++) {
        var base = getBaseCourse(allTimes[i].course)
        if (courses.indexOf(base) === -1) {
            courses.push(base)
        }
    }
    courses.sort()

    courseSelect.innerHTML = '<option value="">All Courses</option>'
    for (var i = 0; i < courses.length; i++) {
        var option = document.createElement("option")
        option.value = courses[i]
        option.textContent = courses[i]
        if (courses[i] === currentValue) {
            option.selected = true
        }
        courseSelect.appendChild(option)
    }
}

function displayTimes() {
    var courseFilter = document.getElementById("course").value
    var filtered = []

    for (var i = 0; i < allTimes.length; i++) {
        var base = getBaseCourse(allTimes[i].course)
        if (courseFilter === "" || base === courseFilter) {
            filtered.push(allTimes[i])
        }
    }

    if (filtered.length === 0) {
        document.getElementById("results").innerHTML = '<div class="empty"><div class="empty-icon">⛳</div>No tee times available for this date.</div>'
        document.getElementById("count").textContent = ""
    } else {
        var html = "<div class='table-wrap'><table>"
        html += "<tr><th>Time</th><th>Course</th><th>Openings</th><th>Holes</th><th>Price</th><th></th></tr>"

        for (var i = 0; i < filtered.length; i++) {
            var t = filtered[i]

            var openClass = "openings-full"
            if (t.openings === 0) openClass = "openings-none"
            else if (t.openings <= 1) openClass = "openings-low"

            var holesClass = "holes-cell"
            if (t.holes === "9") holesClass = "holes-cell holes-9"

            html += "<tr>"
            html += "<td class='time-cell'>" + t.time + "</td>"
            html += "<td class='course-cell'>" + t.course + "</td>"
            html += "<td class='openings-cell " + openClass + "'><svg class='openings-icon' viewBox='0 0 24 24' fill='currentColor'><circle cx='12' cy='7' r='4'/><path d='M12 13c-4.42 0-8 1.79-8 4v2h16v-2c0-2.21-3.58-4-8-4z'/></svg>" + t.openings + " / 4</td>"
            html += "<td><span class='" + holesClass + "'>" + t.holes + " holes</span></td>"
            html += "<td class='price-cell'>$" + t.price + "</td>"
            html += "<td><a href='" + t.bookingUrl + "' target='_blank' class='book-link'>Book</a></td>"
            html += "</tr>"
        }

        html += "</table></div>"

        // Mobile card layout
        html += "<div class='mobile-cards'>"
        for (var i = 0; i < filtered.length; i++) {
            var t = filtered[i]

            var mOpenClass = "openings-full"
            if (t.openings === 0) mOpenClass = "openings-none"
            else if (t.openings <= 1) mOpenClass = "openings-low"

            var mHolesClass = "holes-cell"
            if (t.holes === "9") mHolesClass = "holes-cell holes-9"

            html += "<div class='mobile-tt'>"
            html += "<div class='mobile-tt-left'>"
            html += "<div class='mobile-tt-time'>" + t.time + "</div>"
            html += "<div class='mobile-tt-course'>" + t.course + "</div>"
            html += "<div class='mobile-tt-meta'>"
            html += "<span class='" + mOpenClass + "'><svg class='openings-icon' viewBox='0 0 24 24' fill='currentColor'><circle cx='12' cy='7' r='4'/><path d='M12 13c-4.42 0-8 1.79-8 4v2h16v-2c0-2.21-3.58-4-8-4z'/></svg>" + t.openings + "/4</span>"
            html += " · "
            html += "<span class='" + mHolesClass + "'>" + t.holes + "h</span>"
            html += "</div>"
            html += "</div>"
            html += "<div class='mobile-tt-right'>"
            html += "<div class='mobile-tt-price'>$" + t.price + "</div>"
            html += "<a href='" + t.bookingUrl + "' target='_blank' class='mobile-tt-book'>Book</a>"
            html += "</div>"
            html += "</div>"
        }
        html += "</div>"
        document.getElementById("results").innerHTML = html
        document.getElementById("count").textContent = filtered.length + " tee times available"
    }

    updateAlertSection()
}

function updateAlertSection() {
    var courseFilter = document.getElementById("course").value
    var date = document.getElementById("date").value
    var alertPrompt = document.getElementById("alertPrompt")
    var alertForm = document.getElementById("alertForm")
    var alertContext = document.getElementById("alertContext")
    var message = document.getElementById("message")

    if (courseFilter === "") {
        alertPrompt.style.display = "block"
        alertForm.style.display = "none"
    } else {
        alertPrompt.style.display = "none"
        alertForm.style.display = "block"
        alertContext.textContent = "Get a text when a tee time opens at " + courseFilter + " on " + date + "."
        message.textContent = ""
        message.className = "form-message"
    }
}

async function createAlert() {
    var phone = document.getElementById("phone").value
    var course = document.getElementById("course").value
    var date = document.getElementById("date").value
    var startTime = document.getElementById("startTime").value
    var endTime = document.getElementById("endTime").value
    var message = document.getElementById("message")

    if (!phone) {
        message.textContent = "Please enter your phone number."
        message.className = "form-message form-error"
        return
    }

    var btn = document.getElementById("createBtn")
    btn.disabled = true
    btn.textContent = "Creating..."

    try {
        var response = await fetch("/api/alerts/create", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                phone: phone,
                course: course,
                date: date,
                startTime: startTime,
                endTime: endTime
            })
        })

        var data = await response.json()

        if (!response.ok) {
            message.textContent = data.error || "Failed to create alert."
            message.className = "form-message form-error"
        } else {
            message.textContent = "✓ Alert created! We'll text you when a tee time opens up."
            message.className = "form-message form-success"
            document.getElementById("phone").value = ""
        }
    } catch (err) {
        message.textContent = "Failed to create alert. Please try again."
        message.className = "form-message form-error"
    }

    btn.disabled = false
    btn.textContent = "Create Alert"
}

document.getElementById("date").addEventListener("change", fetchTimes)
document.getElementById("course").addEventListener("change", displayTimes)
document.getElementById("createBtn").addEventListener("click", createAlert)

fetchTimes()
