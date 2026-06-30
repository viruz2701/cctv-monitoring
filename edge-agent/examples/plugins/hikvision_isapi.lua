-- Hikvision ISAPI Protocol Plugin
--
-- Implements device information retrieval via Hikvision ISAPI protocol.
-- ISAPI uses Digest Auth + XML responses, typical for CCTV cameras.
--
-- Usage:
--   local hikvision = require("hikvision_isapi")
--   local info = hikvision.get_device_info(agent, {
--       ip = "192.168.1.100",
--       port = 80,
--       username = "admin",
--       password = "password"
--   })
--
-- Compliance:
--   IEC 62443-3-3 SL-3: Digest auth via agent.http_get
--   OWASP ASVS V5: Input validation via Lua API wrapper

local plugin = {}

-- Device information cache (TTL: 60 seconds)
local cache = {}
local cacheTTL = 60

--- Fetch device information from Hikvision ISAPI.
-- @param agent  The agent API table
-- @param device Table with ip, port, username, password, [timeout]
-- @return Table with model, serial, firmware or nil, error
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

    local cacheKey = device.ip .. ":" .. (device.port or 80)
    local cached = cache[cacheKey]
    if cached and (os.time() - cached.time) < cacheTTL then
        return cached.data
    end

    local port = device.port or 80
    local timeout = device.timeout or 10
    local url = string.format("http://%s:%d/ISAPI/System/deviceInfo", device.ip, port)
    local opts = {
        auth = {
            type = "digest",
            username = device.username,
            password = device.password
        },
        timeout = timeout
    }

    local response, err = agent.http_get(url, opts)
    if not response then
        return nil, "ISAPI request failed: " .. (err or "unknown error")
    end

    local xml = agent.xml_parse(response)
    if not xml then
        return nil, "Failed to parse ISAPI XML response"
    end

    local result = {
        model = xml.modelName,
        serial = xml.serialNumber,
        firmware = xml.firmwareVersion
    }

    -- Update cache
    cache[cacheKey] = {
        data = result,
        time = os.time()
    }

    return result
end

--- Get the number of channels from Hikvision NVR/DVR.
-- @param agent  The agent API table
-- @param device Table with ip, port, username, password
-- @return Number of channels or nil, error
function plugin.get_channel_count(agent, device)
    if not device or not device.ip then
        return nil, "device.ip is required"
    end

    local port = device.port or 80
    local url = string.format("http://%s:%d/ISAPI/System/capabilities", device.ip, port)
    local opts = {
        auth = {
            type = "digest",
            username = device.username,
            password = device.password
        }
    }

    local response, err = agent.http_get(url, opts)
    if not response then
        return nil, "ISAPI capabilities request failed: " .. (err or "unknown error")
    end

    local xml = agent.xml_parse(response)
    if xml and xml.nvrCapabilities and xml.nvrCapabilities.channelCount then
        return tonumber(xml.nvrCapabilities.channelCount)
    end
    return 1
end

--- Get detailed status for all channels.
-- @param agent  The agent API table
-- @param device Table with ip, port, username, password
-- @return Table of channel statuses or nil, error
function plugin.get_channels_status(agent, device)
    if not device or not device.ip then
        return nil, "device.ip is required"
    end

    local port = device.port or 80
    local url = string.format("http://%s:%d/ISAPI/System/Video/inputs/channels", device.ip, port)
    local opts = {
        auth = {
            type = "digest",
            username = device.username,
            password = device.password
        }
    }

    local response, err = agent.http_get(url, opts)
    if not response then
        return nil, "ISAPI channels request failed: " .. (err or "unknown error")
    end

    local xml = agent.xml_parse(response)
    local channels = {}

    -- Parse channel list from XML
    if xml and xml.VideoInputChannel then
        for i = 1, #xml.VideoInputChannel do
            local ch = xml.VideoInputChannel[i]
            table.insert(channels, {
                id = tonumber(ch.id),
                name = ch.name,
                enabled = (ch.enabled == "true"),
                resolution = ch.resolution
            })
        end
    end

    return channels
end

return plugin
