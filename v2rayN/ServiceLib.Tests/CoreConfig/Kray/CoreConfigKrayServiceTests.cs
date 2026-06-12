using AwesomeAssertions;
using ServiceLib.Handler;
using ServiceLib.Models.Dto;
using ServiceLib.Services.CoreConfig;
using Xunit;

namespace ServiceLib.Tests.CoreConfig.Kray;

public class CoreConfigKrayServiceTests
{
    [Fact]
    public void GenerateClientConfigContent_ShouldGenerateNativeKrayConfig()
    {
        var config = CoreConfigTestFactory.CreateConfig(ECoreType.kray);
        CoreConfigTestFactory.BindAppManagerConfig(config);
        var node = CoreConfigTestFactory.CreateKrayNode();
        var context = CoreConfigTestFactory.CreateContext(config, node, ECoreType.kray);

        var result = new CoreConfigKrayService(context).GenerateClientConfigContent();

        result.Success.Should().BeTrue($"ret msg: {result.Msg}");
        result.Data.Should().NotBeNull();

        var root = JsonUtils.ParseJson(result.Data!.ToString())!.AsObject();
        root["auto_start"]!.GetValue<bool>().Should().BeTrue();
        root["active_profile_id"]!.GetValue<string>().Should().Be(node.IndexId);
        root["local"]!["socks_address"]!.GetValue<string>().Should().Be("127.0.0.1:10808");

        var profile = root["profiles"]!.AsArray()[0]!.AsObject();
        profile["transport"]!.GetValue<string>().Should().Be("websocket");
        profile["endpoint"]!.GetValue<string>().Should().Be("kray.example:8443");
        profile["client_id"]!.GetValue<string>().Should().Be(node.Username);
        profile["client_secret"]!.GetValue<string>().Should().Be(node.Password);
        profile["server_public_key"]!.GetValue<string>().Should().Be(node.PublicKey);
        profile["server_signing_key"]!.GetValue<string>().Should().Be(node.PublicKey);
        profile["headers"]!["Host"]!.GetValue<string>().Should().Be("edge.example");
        profile["handshake_padding"]!["min"]!.GetValue<int>().Should().Be(8);
        profile["handshake_padding"]!["max"]!.GetValue<int>().Should().Be(32);
    }

    [Fact]
    public void GenerateClientConfigContent_WithSpeedtestPorts_ShouldUseDedicatedLocalListeners()
    {
        var config = CoreConfigTestFactory.CreateConfig(ECoreType.kray);
        CoreConfigTestFactory.BindAppManagerConfig(config);
        var node = CoreConfigTestFactory.CreateKrayNode();
        var context = CoreConfigTestFactory.CreateContext(config, node, ECoreType.kray);

        var result = new CoreConfigKrayService(context).GenerateClientConfigContent(11223, 19727);

        result.Success.Should().BeTrue($"ret msg: {result.Msg}");
        var root = JsonUtils.ParseJson(result.Data!.ToString())!.AsObject();
        root["local"]!["socks_address"]!.GetValue<string>().Should().Be("127.0.0.1:11223");
        root["local"]!["api_address"]!.GetValue<string>().Should().Be("127.0.0.1:19727");
    }

    [Fact]
    public void GenerateClientConfigContent_TunEnabled_ShouldPassKrayTunConfig()
    {
        var config = CoreConfigTestFactory.CreateConfig(ECoreType.kray);
        config.TunModeItem.EnableTun = true;
        config.TunModeItem.AutoRoute = true;
        config.TunModeItem.StrictRoute = true;
        config.TunModeItem.Mtu = 9000;
        config.TunModeItem.Stack = "gvisor";
        config.TunModeItem.RouteExcludeAddress = ["10.0.0.1/32"];
        CoreConfigTestFactory.BindAppManagerConfig(config);
        var node = CoreConfigTestFactory.CreateKrayNode();
        var context = CoreConfigTestFactory.CreateContext(config, node, ECoreType.kray);

        var result = new CoreConfigKrayService(context).GenerateClientConfigContent();

        result.Success.Should().BeTrue($"ret msg: {result.Msg}");
        var root = JsonUtils.ParseJson(result.Data!.ToString())!.AsObject();
        var tun = root["local"]!["tun"]!.AsObject();
        tun["enabled"]!.GetValue<bool>().Should().BeTrue();
        tun["interface_name"]!.GetValue<string>().Should().Be("krayn_tun");
        tun["mtu"]!.GetValue<int>().Should().Be(9000);
        tun["auto_route"]!.GetValue<bool>().Should().BeTrue();
        tun["strict_route"]!.GetValue<bool>().Should().BeTrue();
        tun["stack"]!.GetValue<string>().Should().Be("gvisor");
        tun["dns_hijack"]!.GetValue<bool>().Should().BeTrue();
        tun["route_exclude"]!.AsArray().Select(x => x!.GetValue<string>()).Should().Contain("10.0.0.1/32");
    }

    [Fact]
    public async Task GenerateClientSpeedtestConfig_ShouldAllowKrayRealPing()
    {
        var config = CoreConfigTestFactory.CreateConfig(ECoreType.kray);
        CoreConfigTestFactory.BindAppManagerConfig(config);
        var node = CoreConfigTestFactory.CreateKrayNode();
        var context = CoreConfigTestFactory.CreateContext(config, node, ECoreType.kray);
        var testItem = new ServerTestItem
        {
            IndexId = node.IndexId,
            Address = node.Address,
            Port = node.Port,
            ConfigType = node.ConfigType,
            QueueNum = 0,
            Profile = node,
            CoreType = ECoreType.kray,
        };
        var fileName = Path.Combine(Path.GetTempPath(), $"{Guid.NewGuid():N}.json");
        try
        {
            var result = await CoreConfigHandler.GenerateClientSpeedtestConfig(config, context, testItem, fileName);

            result.Success.Should().BeTrue($"ret msg: {result.Msg}");
            testItem.AllowTest.Should().BeTrue();
            testItem.Port.Should().BeGreaterThan(0);
        }
        finally
        {
            if (File.Exists(fileName))
            {
                File.Delete(fileName);
            }
        }
    }
}
