namespace ServiceLib.Handler.Fmt;

public class KrayFmt : BaseFmt
{
    public static ProfileItem? Resolve(string str, out string msg)
    {
        msg = ResUI.ConfigurationFormatIncorrect;
        try
        {
            var url = Utils.TryUri(str);
            if (url == null)
            {
                return null;
            }

            var query = Utils.ParseQueryString(url.Query);
            var transport = GetQueryDecoded(query, "transport", GetQueryDecoded(query, "type", "tcp"));
            var item = new ProfileItem
            {
                ConfigType = EConfigType.Kray,
                CoreType = ECoreType.kray,
                Address = url.IdnHost,
                Port = url.Port,
                Username = Uri.UnescapeDataString(url.UserInfo ?? string.Empty),
                Password = GetQueryDecoded(query, "client_secret", GetQueryDecoded(query, "secret")),
                PublicKey = GetQueryDecoded(query, "server_public_key", GetQueryDecoded(query, "pbk")),
                Remarks = Utils.UrlDecode(url.GetComponents(UriComponents.Fragment, UriFormat.Unescaped)),
                Network = NormalizeKrayTransport(transport),
                Sni = GetQueryDecoded(query, "sni", GetQueryDecoded(query, "server_name")),
                AllowInsecure = IsTruthy(GetQueryDecoded(query, "skip_tls_verify", GetQueryDecoded(query, "allowInsecure")))
                    ? Global.StringTrue
                    : string.Empty,
            };

            var extra = item.GetProtocolExtra() with
            {
                KrayHeaders = GetQueryDecoded(query, "headers"),
                KrayPaddingMin = ToNullableInt(GetQueryDecoded(query, "padding_min")),
                KrayPaddingMax = ToNullableInt(GetQueryDecoded(query, "padding_max")),
            };
            item.SetProtocolExtra(extra);

            if (item.Remarks.IsNullOrEmpty())
            {
                item.Remarks = $"{item.Address}:{item.Port}";
            }
            if (item.Address.IsNullOrEmpty() || item.Port <= 0 || item.Username.IsNullOrEmpty() || item.Password.IsNullOrEmpty() || item.PublicKey.IsNullOrEmpty())
            {
                return null;
            }

            msg = string.Empty;
            return item;
        }
        catch (Exception ex)
        {
            Logging.SaveLog("KrayFmt", ex);
            return null;
        }
    }

    public static string? ToUri(ProfileItem? item)
    {
        if (item is null)
        {
            return null;
        }

        var query = new Dictionary<string, string>
        {
            ["transport"] = Utils.UrlEncode(item.Network.IsNullOrEmpty() ? "tcp" : item.Network),
            ["client_secret"] = Utils.UrlEncode(item.Password),
            ["server_public_key"] = Utils.UrlEncode(item.PublicKey),
        };
        if (item.Sni.IsNotEmpty())
        {
            query["server_name"] = Utils.UrlEncode(item.Sni);
        }
        if (item.GetAllowInsecure())
        {
            query["skip_tls_verify"] = "1";
        }

        var extra = item.GetProtocolExtra();
        if (extra.KrayHeaders.IsNotEmpty())
        {
            query["headers"] = Utils.UrlEncode(extra.KrayHeaders);
        }
        if (extra.KrayPaddingMin is not null)
        {
            query["padding_min"] = extra.KrayPaddingMin.Value.ToString();
        }
        if (extra.KrayPaddingMax is not null)
        {
            query["padding_max"] = extra.KrayPaddingMax.Value.ToString();
        }

        return ToUri(EConfigType.Kray, item.Address, item.Port, item.Username, query, $"#{Utils.UrlEncode(item.Remarks)}");
    }

    public static List<ProfileItem>? ResolveJsonSubscription(string strData, string? subRemarks)
    {
        if (JsonUtils.ParseJson(strData) is not JsonNode root)
        {
            return null;
        }

        var nodes = root switch
        {
            JsonArray arr => arr,
            JsonObject obj when obj["profiles"] is JsonArray profiles => profiles,
            JsonObject obj when obj["nodes"] is JsonArray nodesArray => nodesArray,
            JsonObject obj when obj["profile"] is JsonObject profile => new JsonArray(profile.DeepClone()),
            JsonObject obj => new JsonArray(obj.DeepClone()),
            _ => null
        };
        if (nodes is null || nodes.Count == 0)
        {
            return null;
        }

        List<ProfileItem> items = [];
        foreach (var node in nodes)
        {
            if (node is not JsonObject obj)
            {
                continue;
            }
            var item = FromJsonObject(obj, subRemarks);
            if (item != null)
            {
                items.Add(item);
            }
        }
        return items.Count > 0 ? items : null;
    }

    private static ProfileItem? FromJsonObject(JsonObject obj, string? subRemarks)
    {
        var endpoint = GetJsonString(obj, "endpoint");
        var host = GetJsonString(obj, "address", "host", "server");
        var port = GetJsonInt(obj, "port");
        if (endpoint.IsNotEmpty())
        {
            if (TrySplitEndpoint(endpoint, out var endpointHost, out var endpointPort))
            {
                host = endpointHost;
                port = endpointPort;
            }
        }

        var name = GetJsonString(obj, "name", "remarks", "remark");
        if (name.IsNullOrEmpty())
        {
            name = host.IsNotEmpty() && port > 0 ? $"{host}:{port}" : subRemarks ?? "Kray";
        }

        var item = new ProfileItem
        {
            ConfigType = EConfigType.Kray,
            CoreType = ECoreType.kray,
            Remarks = name,
            Address = host,
            Port = port,
            Username = GetJsonString(obj, "client_id", "clientId", "username"),
            Password = GetJsonString(obj, "client_secret", "clientSecret", "password"),
            PublicKey = GetJsonString(obj, "server_public_key", "serverPublicKey", "public_key"),
            Network = NormalizeKrayTransport(GetJsonString(obj, "transport", "network", "type")),
            Sni = GetJsonString(obj, "server_name", "serverName", "sni"),
            AllowInsecure = GetJsonBool(obj, "skip_tls_verify", "skipTLSVerify", "allow_insecure", "allowInsecure")
                ? Global.StringTrue
                : string.Empty,
        };
        var headersNode = obj["headers"];
        var headers = headersNode is null ? string.Empty : JsonUtils.Serialize(headersNode, false);
        item.SetProtocolExtra(item.GetProtocolExtra() with
        {
            KrayHeaders = headers,
            KrayPaddingMin = GetJsonIntNullable(obj, "padding_min", "handshake_padding_min"),
            KrayPaddingMax = GetJsonIntNullable(obj, "padding_max", "handshake_padding_max"),
        });

        if (item.Address.IsNullOrEmpty() || item.Port <= 0 || item.Username.IsNullOrEmpty() || item.Password.IsNullOrEmpty() || item.PublicKey.IsNullOrEmpty())
        {
            return null;
        }
        return item;
    }

    private static bool TrySplitEndpoint(string endpoint, out string host, out int port)
    {
        host = string.Empty;
        port = 0;
        var uri = Utils.TryUri(endpoint.Contains("://") ? endpoint : $"tcp://{endpoint}");
        if (uri is null)
        {
            return false;
        }
        host = uri.IdnHost;
        port = uri.Port;
        return host.IsNotEmpty() && port > 0;
    }

    private static string NormalizeKrayTransport(string? transport)
    {
        return (transport ?? string.Empty).Trim().ToLowerInvariant() switch
        {
            "" => "tcp",
            "websocket" => "websocket",
            "ws" => "websocket",
            "httpupgrade" => "http-upgrade",
            "http-upgrade" => "http-upgrade",
            "upgrade" => "http-upgrade",
            "http" => "http-stream",
            "httpstream" => "http-stream",
            "http-stream" => "http-stream",
            "grpc" => "grpc",
            "xhttp" => "xhttp",
            "tls" => "tls",
            _ => "tcp",
        };
    }

    private static bool IsTruthy(string value)
    {
        return value.Equals("1", StringComparison.OrdinalIgnoreCase)
               || value.Equals("true", StringComparison.OrdinalIgnoreCase)
               || value.Equals("yes", StringComparison.OrdinalIgnoreCase);
    }

    private static int? ToNullableInt(string value)
    {
        return int.TryParse(value, out var number) ? number : null;
    }

    private static string GetJsonString(JsonObject obj, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (obj.TryGetPropertyValue(key, out var value) && value is JsonValue jsonValue && jsonValue.TryGetValue(out string? text))
            {
                return text ?? string.Empty;
            }
        }
        return string.Empty;
    }

    private static int GetJsonInt(JsonObject obj, params string[] keys)
    {
        return GetJsonIntNullable(obj, keys) ?? 0;
    }

    private static int? GetJsonIntNullable(JsonObject obj, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (!obj.TryGetPropertyValue(key, out var value) || value is not JsonValue jsonValue)
            {
                continue;
            }
            if (jsonValue.TryGetValue(out int number))
            {
                return number;
            }
            if (jsonValue.TryGetValue(out string? text) && int.TryParse(text, out number))
            {
                return number;
            }
        }
        return null;
    }

    private static bool GetJsonBool(JsonObject obj, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (!obj.TryGetPropertyValue(key, out var value) || value is not JsonValue jsonValue)
            {
                continue;
            }
            if (jsonValue.TryGetValue(out bool boolean))
            {
                return boolean;
            }
            if (jsonValue.TryGetValue(out string? text))
            {
                return IsTruthy(text ?? string.Empty);
            }
        }
        return false;
    }
}
