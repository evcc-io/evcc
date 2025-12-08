import { describe, expect, test } from "vitest";
import { extractDomain } from "./extractDomain";

describe("extractDomain", () => {
  test("extracts domain from URL", () => {
    expect(extractDomain("https://login.example.org/secure")).toBe("example.org");
    expect(extractDomain("https://www.example.com/path")).toBe("example.com");
    expect(extractDomain("http://subdomain.example.org")).toBe("example.org");
  });

  test("returns IPv4 address as-is", () => {
    expect(extractDomain("https://192.168.1.1/path")).toBe("192.168.1.1");
    expect(extractDomain("http://10.0.0.1")).toBe("10.0.0.1");
    expect(extractDomain("https://127.0.0.1:8080")).toBe("127.0.0.1");
  });

  test("returns full IPv6 address (treated as domain)", () => {
    // IPv6 addresses are treated as domains, but have no dots so return full address
    expect(extractDomain("https://[2001:db8::1]/path")).toBe("2001:db8::1");
    expect(extractDomain("http://[::1]:8080")).toBe("::1");
    expect(extractDomain("https://[2001:0db8:85a3:0000:0000:8a2e:0370:7334]")).toBe(
      "2001:db8:85a3::8a2e:370:7334"
    );
  });

  test("throws for invalid URL", () => {
    expect(() => extractDomain("not-a-url")).toThrow();
    expect(() => extractDomain("")).toThrow();
  });

  test("handles single-part domains", () => {
    expect(extractDomain("https://localhost/path")).toBe("localhost");
    expect(extractDomain("http://local")).toBe("local");
  });
});
