namespace ServiceLib.Services.CoreConfig;

public class CoreConfigKrayService(CoreConfigContext context)
{
    private readonly Config _config = context.AppConfig;
    private readonly ProfileItem _node = context.Node;

    public RetResult GenerateClientConfigContent(int? socksPort = null, int? apiPort = null)
    {
        var ret = new RetResult();
        try
        {
            var profileId = _node.IndexId.IsNotEmpty() ? _node.IndexId : Utils.GetGuid(false);
            var profile = new JsonObject
            {
                ["id"] = profileId,
                ["name"] = _node.Remarks.IsNotEmpty() ? _node.Remarks : $"{_node.Address}:{_node.Port}",
                ["transport"] = NormalizeKrayTransport(_node.Network),
                ["endpoint"] = $"{_node.Address}:{_node.Port}",
                ["client_id"] = _node.Username,
                ["client_secret"] = _node.Password,
                ["server_public_key"] = _node.PublicKey,
                ["server_name"] = _node.Sni,
                ["skip_tls_verify"] = _node.GetAllowInsecure(),
            };

            var extra = _node.GetProtocolExtra();
            var headers = ParseHeaders(extra.KrayHeaders);
            if (headers.Count > 0)
            {
                profile["headers"] = headers;
            }
            if (extra.KrayPaddingMin is not null || extra.KrayPaddingMax is not null)
            {
                profile["handshake_padding"] = new JsonObject
                {
                    ["min"] = extra.KrayPaddingMin ?? 0,
                    ["max"] = extra.KrayPaddingMax ?? extra.KrayPaddingMin ?? 0,
                };
            }

            var apiPortValue = apiPort ?? AppManager.Instance.GetLocalPort(EInboundProtocol.api);
            var socksAddress = socksPort is > 0
                ? $"127.0.0.1:{socksPort.Value}"
                : $"127.0.0.1:{AppManager.Instance.GetLocalPort(EInboundProtocol.socks)}";
            var root = new JsonObject
            {
                ["version"] = 1,
                ["auto_start"] = true,
                ["active_profile_id"] = profileId,
                ["local"] = new JsonObject
                {
                    ["api_address"] = $"127.0.0.1:{apiPortValue}",
                    ["socks_address"] = socksAddress,
                    ["allow_lan"] = _config.Inbound.FirstOrDefault()?.AllowLANConn ?? false,
                    ["mode"] = "rule",
                    ["system_proxy_mode"] = "unchanged",
                    ["resolver_type"] = "system",
                },
                ["profiles"] = new JsonArray(profile),
            };

            ret.Success = true;
            ret.Msg = string.Format(ResUI.SuccessfulConfiguration, "");
            ret.Data = JsonUtils.Serialize(root, true, true);
        }
        catch (Exception ex)
        {
            Logging.SaveLog("CoreConfigKrayService", ex);
            ret.Msg = ResUI.FailedGenDefaultConfiguration;
        }
        return ret;
    }

    private static string NormalizeKrayTransport(string? transport)
    {
        return (transport ?? string.Empty).Trim().ToLowerInvariant() switch
        {
            "" => "tcp",
            "raw" => "tcp",
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

    private static JsonObject ParseHeaders(string? raw)
    {
        var headers = new JsonObject();
        if (raw.IsNullOrEmpty())
        {
            return headers;
        }
        if (JsonUtils.ParseJson(raw) is JsonObject obj)
        {
            foreach (var kv in obj)
            {
                headers[kv.Key] = kv.Value?.DeepClone();
            }
            return headers;
        }
        foreach (var part in raw.Split(';', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries))
        {
            var idx = part.IndexOf(':');
            if (idx <= 0)
            {
                continue;
            }
            headers[part[..idx].Trim()] = part[(idx + 1)..].Trim();
        }
        return headers;
    }
}
