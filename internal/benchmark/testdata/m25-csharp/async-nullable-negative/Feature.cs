using System.Threading.Tasks;
namespace Demo;
class Feature {
    async Task<string?> LoadAsync(string? value) {
        await Task.Yield();
        return value;
    }
}
