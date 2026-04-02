
# skills

## agentscope
```
 # Agent Skills
The agent skills are a collection of folds of instructions, scripts, and resources that you can load dynamically to improve performance on specialized tasks. Each agent skill has a `SKILL.md` file in its folder that describes how to use the skill. If you want to use a skill, you MUST read its `SKILL.md` file carefully.
## pdf
Process PDF files - extract text, create PDFs, merge documents. Use when user asks to read PDF, create PDF, or work with PDF files.
Check "E:\workspace\ai\xkt\xkt-agent\skills\pdf/SKILL.md" for how to use this skill
```

- get_agent_skill_prompt()
```py
_DEFAULT_AGENT_SKILL_TEMPLATE = """## {name}
{description}
Check "{dir}/SKILL.md" for how to use this skill"""

def get_agent_skill_prompt(self) -> str | None:
    skill_descriptions = [self._agent_skill_instruction]
    for skill in self.skills.values():
        text = self._agent_skill_template.format(
            name=skill["name"],
            description=skill["description"],
            dir=skill["dir"],
        )
        skill_descriptions.append(text)

    return "\n".join(skill_descriptions)
```
