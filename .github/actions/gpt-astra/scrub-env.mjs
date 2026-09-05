export default async () => ({
	"shell.env": async (_input, output) => {
		// The provider needs the inference key; agent shell commands do not.
		output.env.AZURE_OPENAI_API_KEY = "";
		output.env.ASTRA_PROMPT = "";
	},
});
