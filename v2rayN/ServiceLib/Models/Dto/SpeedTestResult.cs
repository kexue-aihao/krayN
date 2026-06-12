namespace ServiceLib.Models.Dto;

[Serializable]
public class SpeedTestResult
{
    public string? IndexId { get; set; }

    public string? Delay { get; set; }

    public string? Speed { get; set; }

    public string? IpInfo { get; set; }

    public string? Rtt { get; set; }

    public string? Https { get; set; }

    public string? Jitter { get; set; }
}
