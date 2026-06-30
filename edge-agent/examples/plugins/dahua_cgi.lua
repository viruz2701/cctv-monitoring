-- Dahua CGI Protocol Plugin
--
-- Implements device information retrieval via Dahua CGI protocol.
-- Dahua uses Digest Auth + JSON responses (newer firmware) or XML (legacy).
--
-- Usage:
--   local dahua = require("dahua_cgi")
--   local info = dahua.get_device_info(agent, {
--       ip = "192.168.1.101",
--       port = 80,
--       username = "admin",
--       password = "password"
--   })
--
-- Compliance:
--   IEC 62443-3-3 SL-3: Digest auth via agent.http_get
--   OWASP ASVS V5: Input validation via Lua API wrapper

local plugin = {}

-- Device information cache (TTL: 120 seconds)
local cache = {}
local cacheTTL = 120

--- Fetch device information from Dahua CGI.
-- @param agent  The agent API table
-- @param device Table with ip, port, username, password, [timeout]
-- @return Table with model, serial, firmware, deviceType or nil, error
function plugin.get_device_info(agent, device)
    if not device or not device.ip then
        return nil, "device.ip is required"
    end
    if not device.username then
        return nil, "device.username is required"
    end
    if not device.password then
        return nil, "device.password is required"
    end

    local cacheKey = "device_info_" .. device.ip .. ":" .. (device.port or 80)
    local cached = cache[cacheKey]
    if cached and (os.time() - cached.time) < cacheTTL then
        return cached.data
    end

    local port = device.port or 80
    local timeout = device.timeout or 10
    local baseUrl = string.format("http://%s:%d", device.ip, port)
    local opts = {
        auth = {
            type = "digest",
            username = device.username,
            password = device.password
        },
        timeout = timeout
    }

    -- Dahua CGI: get device type and serial
    local url = baseUrl .. "/cgi-bin/magicBox.cgi?action=getDeviceType"
    local response, err = agent.http_get(url, opts)
    if not response then
        return nil, "Dahua deviceType request failed: " .. (err or "unknown error")
    end

    -- Parse Dahua CGI response format: "var=value\nvar2=value2\n"
    local deviceInfo = parseDahuaResponse(response)
    local deviceType = deviceInfo.deviceType or "unknown"

    -- Get serial number
    local serialUrl = baseUrl .. "/cgi-bin/magicBox.cgi?action=getSerialNo"
    local serialResponse, serr = agent.http_get(serialUrl, opts)
    local serial = "unknown"
    if serialResponse then
        local serialInfo = parseDahuaResponse(serialResponse)
        serial = serialInfo.sn or serialInfo.SerialNo or "unknown"
    end

    -- Get software version
    local versionUrl = baseUrl .. "/cgi-bin/magicBox.cgi?action=getSoftwareVersion"
    local versionResponse, verr = agent.http_get(versionUrl, opts)
    local firmware = "unknown"
    if versionResponse then
        local versionInfo = parseDahuaResponse(versionResponse)
        firmware = versionInfo.version or versionInfo.Version or "unknown"
    end

    local result = {
        model = deviceType,
        serial = serial,
        firmware = firmware,
        deviceType = deviceType,
        manufacturer = "Dahua"
    }

    -- Update cache
    cache[cacheKey] = {
        data = result,
        time = os.time()
    }

    return result
end

--- Get video stream parameters from Dahua camera.
-- @param agent  The agent API table
-- @param device Table with ip, port, username, password
-- @param channel Channel number (default: 1)
-- @param stream  Stream type: "Main", "Extra1", "Extra2", "Extra3" (default: "Main")
-- @return Table with stream parameters or nil, error
function plugin.get_stream_info(agent, device, channel, stream)
    if not device or not device.ip then
        return nil, "device.ip is required"
    end

    local ch = channel or 1
    local st = stream or "Main"
    local port = device.port or 80
    local url = string.format("http://%s:%d/cgi-bin/encode.cgi?action=getCaps&channel=%d&streamType=%s",
        device.ip, port, ch - 1, st)  -- Dahua uses 0-based channels
    local opts = {
        auth = {
            type = "digest",
            username = device.username,
            password = device.password
        }
    }

    local response, err = agent.http_get(url, opts)
    if not response then
        return nil, "Dahua stream info request failed: " .. (err or "unknown error")
    end

    local info = parseDahuaResponse(response)
    local result = {
        channel = ch,
        streamType = st,
        width = tonumber(info.width or info.Width or 0),
        height = tonumber(info.height or info.Height or 0),
        fps = tonumber(info.fps or info.FPS or 0),
        bitrate = tonumber(info.bitRate or info.BitRate or 0)
    }

    return result
end

--- Get events from Dahua device.
-- @param agent     The agent API table
-- @param device    Table with ip, port, username, password
-- @param eventCode Dahua event code (e.g., "VideoMotion", "VideoLoss")
-- @return Table of events or nil, error
function plugin.get_events(agent, device, eventCode)
    if not device or not device.ip then
        return nil, "device.ip is required"
    end

    local port = device.port or 80
    local code = eventCode or "All"
    local url = string.format("http://%s:%d/cgi-bin/eventManager.cgi?action=attach&codes=[%s]",
        device.ip, port, code)
    local opts = {
        auth = {
            type = "digest",
            username = device.username,
            password = device.password
        }
    }

    local response, err = agent.http_get(url, opts)
    if not response then
        return nil, "Dahua events request failed: " .. (err or "unknown error")
    end

    return parseDahuaResponse(response)
end

--- Parse Dahua CGI response format: "var=value\nvar2=value2\n"
-- Some Dahua endpoints return JSON on newer firmwares.
-- @param response Raw response string
-- @return Table with parsed key-value pairs
function parseDahuaResponse(response)
    local result = {}

    -- Try JSON first
    local json, err = agent.json_parse(response)
    if json and type(json) == "table" then
        return json
    end

    -- Fallback to "key=value" parsing
    for line in string.gmatch(response, "[^\r\n]+") do
        local key, value = string.match(line, "^([^=]+)=(.*)$")
        if key and value then
            -- Trim whitespace
            key = string.gsub(key, "^%s*(.-)%s*$", "%1")
            value = string.gsub(value, "^%s*(.-)%s*$", "%1")
            result[key] = value
        end
    end

    return result
end

--- Clear internal cache for a specific device or all devices.
-- @param deviceIp Optional device IP to clear cache for. If nil, clears all.
function plugin.clear_cache(deviceIp)
    if deviceIp then
        for key, _ in pairs(cache) do
            if string.find(key, deviceIp, 1, true) then
                cache[key] = nil
            end
        end
    else
        cache = {}
    end
end

return plugin
