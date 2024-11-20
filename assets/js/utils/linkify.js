export default function linkify(text) {
  const urlRegex = /https?:\/\/[^\s]+/g;

  return text.replace(urlRegex, function (url) {
    return `<a href="${url}" target="_blank">${url}</a>`;
  });
}
