namespace ServiceLib.Handler;

public static class ConnectionHandler
{
    private static readonly string _tag = "ConnectionHandler";
    public static string? LastRealPingError { get; private set; }

    /// <summary>
    /// Runs ping and IP checks and returns a formatted result string.
    /// </summary>
    public static async Task<string> RunAvailabilityCheck()
    {
        var time = await GetRealPingTimeInfo();
        var ip = time > 0 ? await GetIPInfo() : Global.None;

        return string.Format(ResUI.TestMeOutput, time, ip);
    }

    /// <summary>
    /// Gets IP information using the default local proxy.
    /// </summary>
    private static async Task<string?> GetIPInfo()
    {
        var webProxy = await GetWebProxy();

        var ipInfo = await GetIPInfo(webProxy);
        return ipInfo?.ToString() ?? Global.None;
    }

    /// <summary>
    /// Measures real ping time using configured test URL.
    /// </summary>
    private static async Task<int> GetRealPingTimeInfo()
    {
        var responseTime = -1;
        try
        {
            var webProxy = await GetWebProxy();

            for (var i = 0; i < 2; i++)
            {
                responseTime = await GetRealPingTime(webProxy, 10);
                if (responseTime > 0)
                {
                    break;
                }
                await Task.Delay(500);
            }
        }
        catch (Exception ex)
        {
            Logging.SaveLog(_tag, ex);
            return -1;
        }
        return responseTime;
    }

    /// <summary>
    /// Creates local SOCKS proxy instance.
    /// </summary>
    private static async Task<WebProxy?> GetWebProxy()
    {
        var port = AppManager.Instance.GetLocalPort(EInboundProtocol.socks);
        return new WebProxy($"socks5://{Global.Loopback}:{port}");
    }

    /// <summary>
    /// Measures response time by sending HTTP requests through proxy.
    /// </summary>
    public static async Task<int> GetRealPingTime(IWebProxy? webProxy, int downloadTimeout)
    {
        var url = AppManager.Instance.Config.SpeedTestItem.SpeedPingTestUrl;
        var responseTime = -1;
        LastRealPingError = null;
        try
        {
            using var cts = new CancellationTokenSource();
            cts.CancelAfter(TimeSpan.FromSeconds(downloadTimeout));
            using var client = new HttpClient(new SocketsHttpHandler()
            {
                Proxy = webProxy,
                UseProxy = webProxy != null
            });

            List<int> oneTime = [];
            for (var i = 0; i < 2; i++)
            {
                var timer = Stopwatch.StartNew();
                await client.GetAsync(url, cts.Token).ConfigureAwait(false);
                timer.Stop();
                oneTime.Add((int)timer.Elapsed.TotalMilliseconds);
                await Task.Delay(100, cts.Token);
            }
            responseTime = oneTime.Where(x => x > 0).OrderBy(x => x).FirstOrDefault();
        }
        catch (Exception ex)
        {
            LastRealPingError = ex.Message;
            Logging.SaveLog(_tag, ex);
        }
        return responseTime;
    }

    /// <summary>
    /// Gets IP and country information through specified proxy.
    /// </summary>
    public static async Task<IpInfoResult?> GetIPInfo(IWebProxy? webProxy)
    {
        var urls = new List<string>();
        var configuredUrl = AppManager.Instance.Config.SpeedTestItem.IPAPIUrl;
        if (configuredUrl.IsNotEmpty())
        {
            urls.Add(configuredUrl);
        }
        urls.AddRange(Global.IPAPIUrls.Where(url => url.IsNotEmpty()));

        foreach (var url in urls.Distinct(StringComparer.OrdinalIgnoreCase))
        {
            var ipInfo = await GetIPInfoFromUrl(webProxy, url);
            if (ipInfo?.Ip is not null && Utils.IsIpv4(ipInfo.Value.Ip))
            {
                return ipInfo;
            }
        }
        return null;
    }

    private static async Task<IpInfoResult?> GetIPInfoFromUrl(IWebProxy? webProxy, string url)
    {
        try
        {
            var downloadHandle = new DownloadService();
            var result = await downloadHandle.TryDownloadString(url, webProxy, "");
            if (result == null)
            {
                return null;
            }

            var ipInfo = JsonUtils.Deserialize<IPAPIInfo>(result);
            if (ipInfo == null)
            {
                return null;
            }

            var ip = ipInfo.ip ?? ipInfo.clientIp ?? ipInfo.ip_addr ?? ipInfo.query ?? ipInfo.ipAddress ?? ipInfo.address;
            var country = ipInfo.country_code ?? ipInfo.country ?? ipInfo.countryCode ?? ipInfo.country_name ?? ipInfo.location?.country_code ?? "unknown";
            return new IpInfoResult(country, ip);
        }
        catch
        {
            return null;
        }
    }
}
