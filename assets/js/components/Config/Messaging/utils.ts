function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export function formId(serviceType: string, fieldName: string): string {
  return `messagingService${capitalize(serviceType)}${capitalize(fieldName)}`;
}
