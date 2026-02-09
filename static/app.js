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
    if (name.indexOf("McCormick Ranch") === 0) return "McCormick Ranch"
    if (name.indexOf("TPC Scottsdale") === 0) return "TPC Scottsdale"
    if (name.indexOf("Verrado") === 0) return "Verrado"
    if (name.indexOf("Grayhawk") === 0) return "Grayhawk"
    if (name.indexOf("Coyote Lakes") === 0) return "Coyote Lakes"
    if (name.indexOf("Granite Falls") === 0) return "Granite Falls"
    if (name.indexOf("Wigwam") === 0) return "Wigwam"
    if (name.indexOf("Troon North") === 0) return "Troon North"
    if (name.indexOf("Aguila") === 0) return "Aguila"
    if (name.indexOf("Encanto") === 0) return "Encanto"
    if (name.indexOf("AZ Biltmore") === 0) return "AZ Biltmore"
    return name.replace(" Back Nine", "")
}

async function fetchTimes() {
    var date = document.getElementById("date").value
    document.getElementById("results").innerHTML = '<div class="loading"><span class="loading-spinner"></span>Loading tee times...</div>'
    document.getElementById("count").textContent = ""

    try {
        var response = await fetch("/" + METRO + "/teetimes?date=" + date)
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

    updateCityFilter()
}

function updateCityFilter() {
    var citySelect = document.getElementById("city")
    var currentValue = citySelect.value
    var cities = []

    for (var i = 0; i < allTimes.length; i++) {
        if (allTimes[i].city && cities.indexOf(allTimes[i].city) === -1) {
            cities.push(allTimes[i].city)
        }
    }
    cities.sort()

    citySelect.innerHTML = '<option value="">All Cities</option>'
    for (var i = 0; i < cities.length; i++) {
        var option = document.createElement("option")
        option.value = cities[i]
        option.textContent = cities[i]
        if (cities[i] === currentValue) {
            option.selected = true
        }
        citySelect.appendChild(option)
    }
}

function parseTimeToHours(timeStr) {
    var parts = timeStr.match(/(\d+):(\d+)\s*(AM|PM)/i)
    if (!parts) return 0
    var h = parseInt(parts[1])
    var m = parseInt(parts[2])
    var ampm = parts[3].toUpperCase()
    if (ampm === "PM" && h !== 12) h += 12
    if (ampm === "AM" && h === 12) h = 0
    return h + m / 60
}

function formatSliderHour(h) {
    if (h === 0 || h === 12) {
        return (h === 0 ? 12 : 12) + ":00 " + (h < 12 ? "AM" : "PM")
    }
    return (h > 12 ? h - 12 : h) + ":00 " + (h < 12 ? "AM" : "PM")
}

function updateSlider() {
    var fromEl = document.getElementById("timeFrom")
    var toEl = document.getElementById("timeTo")
    var fromVal = parseInt(fromEl.value)
    var toVal = parseInt(toEl.value)

    if (fromVal > toVal) {
        var tmp = fromVal
        fromVal = toVal
        toVal = tmp
        fromEl.value = fromVal
        toEl.value = toVal
    }

    document.getElementById("timeDisplay").textContent = formatSliderHour(fromVal) + " – " + formatSliderHour(toVal)

    var min = parseInt(fromEl.min)
    var max = parseInt(fromEl.max)
    var pctFrom = ((fromVal - min) / (max - min)) * 100
    var pctTo = ((toVal - min) / (max - min)) * 100
    document.getElementById("sliderRange").style.left = pctFrom + "%"
    document.getElementById("sliderRange").style.right = (100 - pctTo) + "%"

    displayTimes()
}

function displayTimes() {
    var courseFilter = document.getElementById("course").value
    var cityFilter = document.getElementById("city").value
    var fromVal = parseInt(document.getElementById("timeFrom").value)
    var toVal = parseInt(document.getElementById("timeTo").value)
    if (fromVal > toVal) { var tmp = fromVal; fromVal = toVal; toVal = tmp }
    var openingsFilter = document.getElementById("openings").value
    var holesFilter = document.getElementById("holes").value
    var filtered = []

    for (var i = 0; i < allTimes.length; i++) {
        var t = allTimes[i]
        var base = getBaseCourse(t.course)
        if (courseFilter !== "" && base !== courseFilter) continue
        if (cityFilter !== "" && t.city !== cityFilter) continue

        var h = parseTimeToHours(t.time)
        if (h < fromVal || h >= toVal) continue

        if (openingsFilter !== "" && t.openings < parseInt(openingsFilter)) continue
        if (holesFilter !== "" && t.holes !== holesFilter) continue

        filtered.push(t)
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
            html += "<td class='course-cell'>" + t.course + "<span class='course-city'>" + (t.city || "") + "</span></td>"
            html += "<td class='openings-cell " + openClass + "'><svg class='openings-icon' viewBox='0 0 24 24' fill='currentColor'><circle cx='12' cy='7' r='4'/><path d='M12 13c-4.42 0-8 1.79-8 4v2h16v-2c0-2.21-3.58-4-8-4z'/></svg>" + t.openings + " / 4</td>"
            html += "<td><span class='" + holesClass + "'>" + t.holes + " holes</span></td>"
            html += "<td class='price-cell'>$" + Math.round(t.price) + "</td>"
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
            html += "<div class='mobile-tt-course'>" + t.course + "<span class='course-city'>" + (t.city || "") + "</span></div>"
            html += "<div class='mobile-tt-meta'>"
            html += "<span class='" + mOpenClass + "'><svg class='openings-icon' viewBox='0 0 24 24' fill='currentColor'><circle cx='12' cy='7' r='4'/><path d='M12 13c-4.42 0-8 1.79-8 4v2h16v-2c0-2.21-3.58-4-8-4z'/></svg>" + t.openings + "/4</span>"
            html += " · "
            html += "<span class='" + mHolesClass + "'>" + t.holes + "h</span>"
            html += "</div>"
            html += "</div>"
            html += "<div class='mobile-tt-right'>"
            html += "<div class='mobile-tt-price'>$" + Math.round(t.price) + "</div>"
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
document.getElementById("city").addEventListener("change", displayTimes)
document.getElementById("timeFrom").addEventListener("input", updateSlider)
document.getElementById("timeTo").addEventListener("input", updateSlider)
document.getElementById("openings").addEventListener("change", displayTimes)
document.getElementById("holes").addEventListener("change", displayTimes)
document.getElementById("createBtn").addEventListener("click", createAlert)

updateSlider()
fetchTimes()
