# Cycle de développement

## Spec Kit

Chaque fonctionnalité suit le cycle [Spec Kit](https://github.com/github/spec-kit),
dont les documents sont rangés dans
[`specs/`](https://github.com/Balsakup/frigg/tree/main/specs), un dossier
par fonctionnalité :

1. **Spécification** (`spec.md`) : ce que veulent les utilisateurs et
   pourquoi, sans détail technique ;
2. **Plan** (`plan.md`, `research.md`, `contracts/`…) : les choix techniques,
   avec une vérification de conformité à la constitution ;
3. **Tâches** (`tasks.md`) : la liste ordonnée des tâches ;
4. **Implémentation**.

La [constitution](https://github.com/Balsakup/frigg/blob/main/.specify/memory/constitution.md)
prévaut sur toute autre pratique du projet.

## Conventions de langue

Frigg est destinée aux associations françaises ; aucune traduction n'est
prévue.

- **Anglais** pour les identifiants : variables, fonctions, types, fichiers,
  tables et colonnes, chemins et champs JSON de l'API, valeurs
  d'énumération, clés de configuration et de journal.
- **Français** pour tout le reste : commentaires, textes de l'interface,
  e-mails, messages d'erreur de l'API et des journaux, sorties des commandes,
  descriptions des tests, documentation et spécifications.

## Commits

Rédigez vos messages de commit en français : un [gitmoji](https://gitmoji.dev)
suivi d'un titre court à l'indicatif, par exemple :

```
🔒 Limite les essais du mot de passe actuel
```
