#!/usr/bin/env tsx

import { readFileSync, readdirSync, existsSync } from "fs";
import { join, dirname, basename, extname } from "path";
import { fileURLToPath } from "url";

interface TranslationObject {
  [key: string]: string | TranslationObject;
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Extract placeholders from a translation string (e.g., "{title}", "{soc}", etc.)
function extractPlaceholders(text: string): string[] {
  const matches = text.match(/\{[^}]+\}/g);
  if (!matches) return [];
  
  // Remove duplicates and sort
  const unique = [...new Set(matches)];
  return unique.sort();
}

// Recursively collect all translation keys and their placeholders from nested objects
function collectTranslations(obj: TranslationObject, prefix = ""): Map<string, string[]> {
  const result = new Map<string, string[]>();
  
  for (const [key, value] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${key}` : key;
    
    if (typeof value === "string") {
      result.set(fullKey, extractPlaceholders(value));
    } else if (typeof value === "object" && value !== null) {
      const nested = collectTranslations(value, fullKey);
      for (const [nestedKey, placeholders] of nested) {
        result.set(nestedKey, placeholders);
      }
    }
  }
  
  return result;
}

// Load and parse JSON translation file
function loadTranslation(filePath: string): Map<string, string[]> {
  try {
    const content = readFileSync(filePath, "utf-8");
    const parsed = JSON.parse(content) as TranslationObject;
    return collectTranslations(parsed);
  } catch (error) {
    console.error(`Error loading ${filePath}:`, error);
    process.exit(1);
  }
}

// Get all translation files in the directory
function getTranslationFiles(): string[] {
  const files = readdirSync(__dirname)
    .filter(file => extname(file) === ".json")
    .map(file => join(__dirname, file));
  
  return files;
}

// Check if two placeholder arrays are equivalent
function placeholdersMatch(source: string[], target: string[]): boolean {
  if (source.length !== target.length) return false;
  return source.every((placeholder, index) => placeholder === target[index]);
}

// Main validation function
function validateTranslations(): void {
  console.log("🔍 Checking translation placeholder consistency...");
  
  const sourceFile = join(__dirname, "en.json");
  
  if (!existsSync(sourceFile)) {
    console.error(`❌ Source file not found: ${sourceFile}`);
    process.exit(1);
  }
  
  const sourceTranslations = loadTranslation(sourceFile);
  
  const translationFiles = getTranslationFiles()
    .filter(file => basename(file) !== "en.json");
  
  if (translationFiles.length === 0) {
    console.log("ℹ️  No translation files to check.");
    process.exit(0);
  }
  
  let hasErrors = false;
  let totalErrors = 0;
  
  for (const file of translationFiles) {
    const lang = basename(file, ".json");
    const targetTranslations = loadTranslation(file);
    
    const fileErrors: Array<{key: string, expected: string, found: string}> = [];
    
    for (const [key, sourcePlaceholders] of sourceTranslations) {
      const targetPlaceholders = targetTranslations.get(key);
      
      // Only check if translation exists AND source has placeholders
      if (targetPlaceholders && sourcePlaceholders.length > 0 && !placeholdersMatch(sourcePlaceholders, targetPlaceholders)) {
        hasErrors = true;
        totalErrors++;
        
        const sourceStr = sourcePlaceholders.join(", ");
        const targetStr = targetPlaceholders.join(", ") || "(none)";
        fileErrors.push({key, expected: sourceStr, found: targetStr});
      }
    }
    
    if (fileErrors.length > 0) {
      console.log(`\n📄 i18n/${lang}.json (${fileErrors.length} errors)`);
      for (const error of fileErrors) {
        console.log(`  ${error.key}`);
        console.log(`    expected: ${error.expected}`);
        console.log(`    found:    ${error.found}`);
      }
    }
  }
  
  if (hasErrors) {
    console.log(`\n💡 ${totalErrors} placeholder errors found.`);
    process.exit(1);
  } else {
    console.log("✅ All translations have correct placeholders!");
    process.exit(0);
  }
}

// Run the validation
validateTranslations();