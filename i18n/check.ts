#!/usr/bin/env tsx

import { readFileSync, readdirSync, existsSync } from "fs";
import { join, dirname, basename, extname, resolve } from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Security: Validate that file path is within the i18n directory
function validateFilePath(filePath: string): string {
  const resolvedPath = resolve(filePath);
  const resolvedBaseDir = resolve(__dirname);
  
  if (!resolvedPath.startsWith(resolvedBaseDir)) {
    throw new Error(`Path traversal attempt detected: ${filePath}`);
  }
  
  // Ensure it's a JSON file
  if (extname(resolvedPath) !== ".json") {
    throw new Error(`Invalid file type: ${filePath}`);
  }
  
  return resolvedPath;
}

// Extract placeholders from a translation string (e.g., "{title}", "{soc}", etc.)
function extractPlaceholders(text: string): string[] {
  const matches = text.match(/\{[^}]+\}/g);
  if (!matches) return [];
  
  // Remove duplicates and sort
  const unique = [...new Set(matches)];
  return unique.sort();
}

// Flatten nested translation object into dot-notation keys
function flattenObject(obj: any, prefix = ""): Record<string, string> {
  return Object.entries(obj).reduce((acc, [key, val]) => {
    const path = prefix ? `${prefix}.${key}` : key;
    if (val && typeof val === "object") {
      Object.assign(acc, flattenObject(val, path));
    } else if (typeof val === "string") {
      acc[path] = val;
    }
    return acc;
  }, {} as Record<string, string>);
}

// Compare placeholder arrays using Set equality for better performance
function placeholdersMatch(a: string[], b: string[]): boolean {
  const setA = new Set(a);
  const setB = new Set(b);
  if (setA.size !== setB.size) return false;
  for (const placeholder of setA) {
    if (!setB.has(placeholder)) return false;
  }
  return true;
}

// Load and parse JSON translation file safely
function loadTranslationFile(filePath: string): Record<string, string> {
  const validatedPath = validateFilePath(filePath);
  const content = readFileSync(validatedPath, "utf-8");
  const parsed = JSON.parse(content);
  return flattenObject(parsed);
}

// Get all translation files in the directory safely
function getTranslationFiles(): string[] {
  const files = readdirSync(__dirname)
    .filter(file => {
      // Only allow valid filename characters and JSON extension
      return /^[a-zA-Z0-9_-]+\.json$/.test(file);
    })
    .map(file => validateFilePath(join(__dirname, file)));
  
  return files;
}

// Main validation function
function validateTranslations(): void {
  console.log("🔍 Checking translation placeholder consistency...");
  
  const sourceFile = validateFilePath(join(__dirname, "en.json"));
  
  if (!existsSync(sourceFile)) {
    console.error(`❌ Source file not found: ${sourceFile}`);
    process.exit(1);
  }
  
  const sourceFlat = loadTranslationFile(sourceFile);
  const translationFiles = getTranslationFiles()
    .filter(file => basename(file) !== "en.json");
  
  if (translationFiles.length === 0) {
    console.log("ℹ️  No translation files to check.");
    process.exit(0);
  }
  
  const allErrors: Array<{file: string; key: string; expected: string[]; found: string[]}> = [];
  
  for (const file of translationFiles) {
    const lang = basename(file, ".json");
    const targetFlat = loadTranslationFile(file);
    
    for (const [key, sourceStr] of Object.entries(sourceFlat)) {
      const sourcePlaceholders = extractPlaceholders(sourceStr);
      
      // Only check if source has placeholders
      if (sourcePlaceholders.length === 0) continue;
      
      // Skip if translation doesn't exist at all (fallback to English is fine)
      if (!(key in targetFlat)) continue;
      
      const targetPlaceholders = extractPlaceholders(targetFlat[key]);
      
      if (!placeholdersMatch(sourcePlaceholders, targetPlaceholders)) {
        allErrors.push({
          file: lang,
          key,
          expected: sourcePlaceholders,
          found: targetPlaceholders
        });
      }
    }
  }
  
  // Group errors by file and output
  const errorsByFile = new Map<string, typeof allErrors>();
  for (const error of allErrors) {
    if (!errorsByFile.has(error.file)) {
      errorsByFile.set(error.file, []);
    }
    errorsByFile.get(error.file)!.push(error);
  }
  
  for (const [lang, errors] of errorsByFile) {
    console.log(`\n📄 i18n/${lang}.json (${errors.length} errors)`);
    for (const error of errors) {
      console.log(`  ${error.key}`);
      console.log(`    expected: ${error.expected.join(", ")}`);
      console.log(`    found:    ${error.found.join(", ") || "(none)"}`);
    }
  }
  
  if (allErrors.length > 0) {
    console.log(`\n💡 ${allErrors.length} placeholder errors found.`);
    process.exit(1);
  } else {
    console.log("✅ All translations have correct placeholders!");
    process.exit(0);
  }
}

// Run the validation
validateTranslations();